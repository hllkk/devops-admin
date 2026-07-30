import { useAuthStore } from '@/store/modules/auth';
import { useDiskStore } from '@/store/modules/disk';
import { request } from '@/service/request';

/**
 * 第1期：网盘只读文件列表 API
 *
 * ⚠️ MOCK 模式：当前后端网盘接口尚未实现，USE_MOCK 硬编码为 true，
 *   fetchGetFileList / fetchResolvePath 返回前端构造的假数据以跑通只读列表。
 *   后端 /file-meta/list、/file-meta/path-resolve 就绪后，将 USE_MOCK 改为 false 即可切换真实接口。
 */
const USE_MOCK = true;

// ============ 类型/图标映射 ============

/** 将 MIME 类型转换为前端文件分类 */
function contentTypeToFileType(contentType: string): string {
  if (!contentType) return 'other';
  const ct = contentType.toLowerCase();
  if (ct.startsWith('image/')) return 'image';
  if (ct.startsWith('video/')) return 'video';
  if (ct.startsWith('audio/')) return 'audio';
  if (ct === 'application/pdf') return 'document';
  if (ct.startsWith('text/')) return 'document';
  if (ct.includes('word') || ct.includes('spreadsheet') || ct.includes('presentation') || ct.includes('document'))
    return 'document';
  return 'other';
}

/** 扩展名 → iconify 图标名 */
export function getFileIcon(ext?: string): string {
  if (!ext) return 'material-symbols:description';
  const lower = ext.toLowerCase().replace(/^\./, '');
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(lower)) return 'material-symbols:image';
  if (['mp4', 'avi', 'mkv', 'mov', 'wmv', 'flv'].includes(lower)) return 'material-symbols:video-camera-back';
  if (['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'ape'].includes(lower)) return 'material-symbols:audiotrack';
  if (['doc', 'docx', 'pdf', 'txt', 'md'].includes(lower)) return 'material-symbols:description';
  if (['xls', 'xlsx', 'csv'].includes(lower)) return 'material-symbols:table-chart';
  if (['ppt', 'pptx'].includes(lower)) return 'material-symbols:slideshow';
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(lower)) return 'material-symbols:folder-zip';
  if (['json', 'js', 'ts', 'vue', 'html', 'css', 'py', 'java', 'go', 'xml', 'yaml', 'yml', 'sql'].includes(lower))
    return 'material-symbols:code';
  return 'material-symbols:draft';
}

/** 将后端 FileListResponse 转换为前端 FileItem 格式 */
export function mapBackendFileList(backendData: Api.Disk.BackendFileListResponse) {
  const list: Api.Disk.FileItem[] = (backendData.list || []).map(item => {
    // 处理扩展名：去掉前导的点号（如 '.md' -> 'md'）
    const cleanExtension = item.extendName ? item.extendName.replace(/^\./, '') : undefined;

    return {
      createBy: '',
      createTime: item.createTime || '',
      updateBy: '',
      updateTime: item.updateTime || '',
      fileId: item.id,
      fileName: item.name,
      fileType: item.isDir ? 'folder' : contentTypeToFileType(item.contentType),
      fileSize: item.size,
      fileExtension: cleanExtension,
      extendName: item.extendName, // 保留原始值供其他组件使用
      parentId: null,
      filePath: item.filePath,
      modifyTime: item.updateTime,
      isFolder: item.isDir,
      icon: item.isDir ? 'material-symbols:folder' : getFileIcon(cleanExtension),
      mediaCover: item.mediaCover || false,
      showCover: item.showCover || false,
      music: item.music,
      video: item.video,
      isFavorite: item.isFavorite || false,
      isShare: item.isShare || false,
      sharedUserCount: item.sharedUserCount || 0,
      sharedDeptCount: item.sharedDeptCount || 0
    };
  });

  return {
    rows: list,
    total: backendData.total || 0
  };
}

// ============ mock 数据（前端联调用，后端就绪后整段可删） ============

const MOCK_FOLDERS = ['文档库', '图片素材', '视频集', '工作资料'] as const;
const MOCK_FILE_PROTO = [
  { name: '风景', ext: 'jpg', ct: 'image/jpeg', size: 2_400_000 },
  { name: '设计稿', ext: 'png', ct: 'image/png', size: 1_800_000 },
  {
    name: '演示文稿',
    ext: 'pptx',
    ct: 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    size: 5_200_000
  },
  { name: '报表', ext: 'xlsx', ct: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', size: 980_000 },
  {
    name: '合同',
    ext: 'docx',
    ct: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    size: 320_000
  },
  { name: '说明书', ext: 'pdf', ct: 'application/pdf', size: 1_500_000 },
  { name: '笔记', ext: 'txt', ct: 'text/plain', size: 12_000 },
  { name: 'README', ext: 'md', ct: 'text/markdown', size: 8_000 },
  { name: '会议录像', ext: 'mp4', ct: 'video/mp4', size: 85_000_000 },
  { name: '背景音乐', ext: 'mp3', ct: 'audio/mpeg', size: 4_600_000 },
  { name: '归档', ext: 'zip', ct: 'application/zip', size: 22_000_000 },
  { name: 'config', ext: 'json', ct: 'application/json', size: 3_400 }
] as const;

let mockItemsCache: Api.Disk.BackendFileItem[] | null = null;

/** 构造 mock 文件树：根目录（文件夹 + 100 个散文件，够测无限滚动）+ 每个文件夹 20 个子文件 */
function getMockItems(): Api.Disk.BackendFileItem[] {
  if (mockItemsCache) return mockItemsCache;

  const items: Api.Disk.BackendFileItem[] = [];
  let id = 1000;
  const baseTime = '2026-07-20T10:00:00';

  const mk = (
    name: string,
    ext: string,
    ct: string,
    size: number,
    dir: string,
    isDir: boolean
  ): Api.Disk.BackendFileItem => ({
    id: String(id++),
    name: ext ? `${name}.${ext}` : name,
    extendName: ext,
    isDir,
    size,
    updateTime: baseTime,
    createTime: baseTime,
    contentType: ct,
    filePath: dir,
    isFavorite: id % 7 === 0,
    isShare: id % 11 === 0,
    sharedUserCount: 0,
    sharedDeptCount: 0,
    userId: 1,
    mediaCover: false,
    showCover: false
  });

  // 根目录：文件夹
  for (const f of MOCK_FOLDERS) items.push(mk(f, '', '', 0, '/', true));
  // 根目录：100 个散文件
  for (let i = 1; i <= 100; i++) {
    const p = MOCK_FILE_PROTO[i % MOCK_FILE_PROTO.length];
    items.push(mk(`${p.name}-${i}`, p.ext, p.ct, p.size + i * 1024, '/', false));
  }
  // 各文件夹：20 个子文件
  for (const f of MOCK_FOLDERS) {
    for (let i = 1; i <= 20; i++) {
      const p = MOCK_FILE_PROTO[(i + 3) % MOCK_FILE_PROTO.length];
      items.push(mk(`${f}-${i}`, p.ext, p.ct, p.size + i * 512, `/${f}`, false));
    }
  }

  mockItemsCache = items;
  return items;
}

/** 根目录文件夹名 → id 映射（restoreFromPath 用） */
function mockFolderNameToId(): Record<string, string> {
  const map: Record<string, string> = {};
  for (const it of getMockItems()) {
    if (it.isDir && it.filePath === '/') map[it.name] = it.id;
  }
  return map;
}

/** mock：按目录/类型/关键词过滤 + 文件夹优先排序 + 分页 */
function mockGetFileList(
  params: Api.Disk.FileSearchParams | undefined,
  currentDirectory: string
): Api.Disk.BackendFileListResponse {
  let list = getMockItems().filter(it => it.filePath === currentDirectory);

  const fileType = params?.fileType && params.fileType !== 'all' ? params.fileType : '';
  if (fileType) {
    list = list.filter(it => !it.isDir && contentTypeToFileType(it.contentType) === fileType);
  }

  const keyword = params?.keyword || '';
  if (keyword) {
    list = list.filter(it => it.name.toLowerCase().includes(keyword.toLowerCase()));
  }

  // 文件夹优先 + 排序
  const field = params?.sortField;
  const order = params?.sortOrder === 'desc' ? -1 : 1;
  list = [...list].sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
    switch (field) {
      case 'name':
        return order * a.name.localeCompare(b.name);
      case 'size':
        return order * (a.size - b.size);
      case 'modifyTime':
        return order * (new Date(a.updateTime).getTime() - new Date(b.updateTime).getTime());
      default:
        return a.name.localeCompare(b.name);
    }
  });

  const total = list.length;
  const pageNum = params?.pageNum || 1;
  const pageSize = params?.pageSize || 50;
  const start = (pageNum - 1) * pageSize;

  return { list: list.slice(start, start + pageSize), total, page: pageNum, size: pageSize };
}

/** mock：按路径段返回面包屑 */
function mockResolvePath(path: string): Api.Disk.PathResolveResponse {
  const segs = path.split('/').filter(Boolean);
  const folderIds = mockFolderNameToId();
  const breadcrumb: Api.Disk.BreadcrumbItem[] = [{ fileId: null, fileName: '根目录', filePath: '/' }];
  let cur = '';
  let lastId: CommonType.IdType = 0;
  let lastName = '根目录';

  for (const seg of segs) {
    cur = `${cur}/${seg}`;
    const fid: CommonType.IdType = folderIds[seg] ?? String(cur.length);
    breadcrumb.push({ fileId: fid, fileName: seg, filePath: cur });
    lastId = fid;
    lastName = seg;
  }

  return { fileId: lastId, fileName: lastName, parentId: null, filePath: path, breadcrumb };
}

/** mock 响应包装：返回与 request 一致的 { data, error } 结构，并模拟网络延迟 */
function mockResponse<T>(data: T, delay = 200): Promise<{ data: T; error: null }> {
  return new Promise(resolve => {
    setTimeout(() => resolve({ data, error: null }), delay);
  });
}

// ============ API ============

/** 获取文件列表 */
export function fetchGetFileList(params?: Api.Disk.FileSearchParams) {
  const diskStore = useDiskStore();

  if (USE_MOCK) {
    const currentDirectory =
      diskStore.currentPath.length > 0 ? `/${diskStore.currentPath.map(item => item.fileName).join('/')}` : '/';
    const data = mockGetFileList(params, currentDirectory);
    return mockResponse(data, 200);
  }

  const userId = Number(useAuthStore().userInfo.user?.userId ?? 0);
  const currentDirectory =
    diskStore.currentPath.length > 0 ? `/${diskStore.currentPath.map(item => item.fileName).join('/')}` : '/';

  const queryType = params?.fileType && params.fileType !== 'all' ? params.fileType : '';

  return request<Api.Disk.BackendFileListResponse>({
    url: '/file-meta/list',
    method: 'get',
    params: {
      userId,
      currentDirectory,
      queryType,
      keyword: params?.keyword || '',
      page: params?.pageNum || 1,
      pageSize: params?.pageSize || 50,
      sortBy: params?.sortField === 'modifyTime' ? 'time' : params?.sortField,
      sortOrder: params?.sortOrder
    }
  });
}

/** 路径解析 - 用于URL导航恢复 */
export function fetchResolvePath(path: string) {
  if (USE_MOCK) {
    const data = mockResolvePath(path);
    return mockResponse(data, 100);
  }

  return request<Api.Disk.PathResolveResponse>({
    url: '/file-meta/path-resolve',
    method: 'get',
    params: { path }
  });
}
