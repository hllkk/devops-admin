/**
 * 网盘文件名同名冲突解决。
 *
 * 同级目录存在同名项时,生成不重名的新名:基名 → 基名(1) → 基名(2)...
 * 保留扩展名(只拆最后一个扩展名段): a.txt → a(1).txt; a.tar.gz → a.tar(1).gz。
 * 大小写不敏感比较(与后端同名检测语义一致)。
 *
 * @param baseName 期望的名字
 * @param existingNames 当前目录已有的名字集合
 */
export function resolveNameConflict(baseName: string, existingNames: string[]): string {
  if (!existingNames.some(n => n.toLowerCase() === baseName.toLowerCase())) return baseName;

  const dot = baseName.lastIndexOf('.');
  const stem = dot > 0 ? baseName.slice(0, dot) : baseName;
  const ext = dot > 0 ? baseName.slice(dot) : ''; // 含前导点

  const lower = existingNames.map(n => n.toLowerCase());
  let i = 1;
  let candidate = `${stem}(${i})${ext}`;
  while (lower.includes(candidate.toLowerCase())) {
    i += 1;
    candidate = `${stem}(${i})${ext}`;
  }
  return candidate;
}

/**
 * 文件夹选择后,从 input.webkitEntries 递归收集所有目录相对路径(含空目录)。
 *
 * webkitdirectory 的 input.files 只返回文件(空目录无 File 项),故无法靠 files 拿到空目录。
 * webkitEntries(Chromium)提供 FileSystemEntry,可递归 readEntries 拿到含空目录的完整目录树。
 * 返回的路径是相对顶层目录的(如 'TopDir/sub'),与 webkitRelativePath 的目录段一致,供 ensure-folders 预建。
 *
 * 降级:webkitEntries 为空(非 Chromium / 拖拽 / 浏览器未填充)时,从 files 的 webkitRelativePath
 * 提取所有中间目录段。此降级**无法覆盖真正的空目录**(无文件的目录),但能预建有文件的中间目录,
 * 避免依赖 merge 阶段懒建时的并发竞态。
 */
export async function collectFolderDirs(input: HTMLInputElement): Promise<string[]> {
  const webkitEntries = (input as HTMLInputElement & { webkitEntries?: FileSystemEntry[] }).webkitEntries;
  if (webkitEntries && webkitEntries.length) {
    const dirs = new Set<string>();
    await Promise.all(webkitEntries.map(entry => traverseEntry(entry, '', dirs)));
    return [...dirs].filter(Boolean);
  }
  // 降级:从 files 的 webkitRelativePath 提取目录段
  const dirs = new Set<string>();
  const files = Array.from(input.files || []);
  for (const f of files) {
    const rel = (f as File & { webkitRelativePath?: string }).webkitRelativePath || '';
    const segs = rel.split('/').slice(0, -1); // 去末段文件名
    for (let i = 1; i <= segs.length; i += 1) {
      const dir = segs.slice(0, i).join('/');
      if (dir) dirs.add(dir);
    }
  }
  return [...dirs].filter(Boolean);
}

/** 递归遍历 FileSystemEntry:directory → 记录路径 + 递归子;file → 忽略(files 用 input.files) */
async function traverseEntry(entry: FileSystemEntry, prefix: string, dirs: Set<string>): Promise<void> {
  if (!entry.isDirectory) return;
  const path = prefix ? `${prefix}/${entry.name}` : entry.name;
  if (path) dirs.add(path);
  const reader = (entry as FileSystemDirectoryEntry).createReader();
  // readEntries 单次最多返回 100 条,需循环读到空数组
  const children: FileSystemEntry[] = [];
  let batch: FileSystemEntry[] = [];
  do {
    batch = await readEntriesP(reader);
    children.push(...batch);
  } while (batch.length > 0);
  await Promise.all(children.map(child => traverseEntry(child, path, dirs)));
}

/** 将回调式 readEntries 包装为 Promise */
function readEntriesP(reader: FileSystemDirectoryReader): Promise<FileSystemEntry[]> {
  return new Promise((resolve, reject) => {
    reader.readEntries(resolve, reject);
  });
}

/**
 * 拖拽落下的 FileSystemEntry 树 → 扁平 {file, relativePath}[] + 目录路径[](含空目录)。
 *
 * 与 collectFolderDirs(input) 的区别:后者从 webkitdirectory 的 input 取数据,浏览器已自动给每个 File
 * 填充 webkitRelativePath;而拖拽(dataTransfer.items[i].webkitGetAsEntry())拿到的 File **没有**
 * webkitRelativePath,必须在递归 entry 树时手动记录相对路径。
 *
 * 调用约束:webkitGetAsEntry() 必须在 drop 事件同步期内、任何 await 之前调完(否则浏览器清空拖拽数据
 * 存储,后续返回 null,导致多文件拖拽只产生一个任务)。本函数接收的已是稳定 entry 引用,可安全异步读取。
 *
 * - file entry → file() 取 File,relativePath = "目录前缀/文件名";顶层文件(prefix 空)relativePath 留空,
 *   与单文件无目录语义一致,uploadFiles 据此归入散文件而非文件夹聚合。
 * - dir entry  → 收集路径(含空目录:readEntries 返回空仍记录),供 ensure-folders 预建目录树。
 */
/** 拖拽落下的截断原因:超数量上限 / 超累计体积上限 */
export type DropTruncated = 'count' | 'size';

/**
 * 拖拽落下的 FileSystemEntry 树 → 扁平 {file, relativePath}[] + 目录路径[](含空目录)。
 *
 * 与 collectFolderDirs(input) 的区别:后者从 webkitdirectory 的 input 取数据,浏览器已自动给每个 File
 * 填充 webkitRelativePath;而拖拽(dataTransfer.items[i].webkitGetAsEntry())拿到的 File **没有**
 * webkitRelativePath,必须在递归 entry 树时手动记录相对路径。
 *
 * 调用约束:webkitGetAsEntry() 必须在 drop 事件同步期内、任何 await 之前调完(否则浏览器清空拖拽数据
 * 存储,后续返回 null,导致多文件拖拽只产生一个任务)。本函数接收的已是稳定 entry 引用,可安全异步读取。
 *
 * - file entry → file() 取 File,relativePath = "目录前缀/文件名";顶层文件(prefix 空)relativePath 留空,
 *   与单文件无目录语义一致,uploadFiles 据此归入散文件而非文件夹聚合。
 * - dir entry  → 收集路径(含空目录:readEntries 返回空仍记录),供 ensure-folders 预建目录树。
 * - limits(可选):递归过程中即时截断,防海量小文件全量读入 OOM;超限即停止收集并返回 truncated 原因。
 */
export async function collectDropEntries(
  entries: FileSystemEntry[],
  limits?: { maxCount?: number; maxSize?: number }
): Promise<{
  files: { file: File; relativePath?: string }[];
  dirs: string[];
  truncated?: DropTruncated;
}> {
  const files: { file: File; relativePath?: string }[] = [];
  const dirs = new Set<string>();
  const state = { count: 0, size: 0, truncated: undefined as DropTruncated | undefined };
  await Promise.all(entries.map(entry => traverseDropEntry(entry, '', files, dirs, limits, state)));
  return { files, dirs: [...dirs].filter(Boolean), truncated: state.truncated };
}

/** 递归遍历拖拽 entry:file → 取 File + 记 relativePath(超 limits 即截断);directory → 记路径(含空目录)+ 递归子 */
async function traverseDropEntry(
  entry: FileSystemEntry,
  prefix: string,
  files: { file: File; relativePath?: string }[],
  dirs: Set<string>,
  limits: { maxCount?: number; maxSize?: number } | undefined,
  state: { count: number; size: number; truncated?: DropTruncated }
): Promise<void> {
  if (state.truncated) return; // 已截断,剩余 entry 跳过
  if (entry.isFile) {
    if (limits?.maxCount && state.count >= limits.maxCount) {
      state.truncated = 'count';
      return;
    }
    const file = await fileEntryToFile(entry as FileSystemFileEntry);
    if (limits?.maxSize && state.size + file.size > limits.maxSize) {
      state.truncated = 'size';
      return;
    }
    state.count += 1;
    state.size += file.size;
    // 顶层文件(prefix 空)relativePath 留空 → uploadFiles 归入散文件,不误判为文件夹
    const relativePath = prefix ? `${prefix}/${entry.name}` : '';
    files.push({ file, relativePath });
    return;
  }
  if (!entry.isDirectory) return;
  const path = prefix ? `${prefix}/${entry.name}` : entry.name;
  if (path) dirs.add(path);
  const reader = (entry as FileSystemDirectoryEntry).createReader();
  // readEntries 单次最多返回 100 条,需循环读到空数组(与 traverseEntry 一致)
  const children: FileSystemEntry[] = [];
  let batch: FileSystemEntry[] = [];
  do {
    batch = await readEntriesP(reader);
    children.push(...batch);
  } while (batch.length > 0);
  // 串行递归子:便于截断时及时 break(并发下截断不及时,海量小文件仍可能瞬间占用内存)
  for (const child of children) {
    if (state.truncated) break;
    await traverseDropEntry(child, path, files, dirs, limits, state);
  }
}

/** 将 FileSystemFileEntry 的回调式 file() 包装为 Promise<File> */
function fileEntryToFile(entry: FileSystemFileEntry): Promise<File> {
  return new Promise((resolve, reject) => {
    entry.file(resolve, reject);
  });
}
