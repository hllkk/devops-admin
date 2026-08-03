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
