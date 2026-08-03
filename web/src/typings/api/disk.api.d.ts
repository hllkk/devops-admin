/**
 * Namespace Api
 *
 * All backend api type
 */
declare namespace Api {
  /**
   * namespace Disk
   *
   * backend api module: "disk"
   *
   * 第1期：只读文件列表所需类型。分享/配额/回收站/上传/版本/挂载等类型后续期补充。
   */
  namespace Disk {
    /** 文件类型枚举 */
    type FileType = 'all' | 'image' | 'document' | 'video' | 'audio' | 'other';

    /**
     * 后端 FileListResponse 原始字段（/file-meta/list 响应项）
     * 对应 backend model/disk/response FileListResponse，JSON tag 一一对应
     */
    type BackendFileItem = {
      /** 文件ID */
      id: string;
      /** 文件名 */
      name: string;
      /** 扩展名（可能含前导点号） */
      extendName: string;
      /** 是否为文件夹 */
      isDir: boolean;
      /** 文件大小（字节） */
      size: number;
      /** 更新时间 */
      updateTime: string;
      /** 创建时间 */
      createTime: string;
      /** MIME 类型 */
      contentType: string;
      /** 文件路径 */
      filePath: string;
      /** 是否已收藏 */
      isFavorite: boolean;
      /** 是否已分享（外链） */
      isShare: boolean;
      /** 已共享给用户数 */
      sharedUserCount: number;
      /** 已共享给部门数 */
      sharedDeptCount: number;
      /** 归属用户ID */
      userId: number;
      /** 是否有媒体封面/缩略图 */
      mediaCover: boolean;
      /** 是否显示封面 */
      showCover: boolean;
      /** 音乐信息 */
      music?: MusicInfo;
      /** 视频信息 */
      video?: VideoInfo;
    };

    /** 后端文件列表响应（/file-meta/list） */
    type BackendFileListResponse = {
      list: BackendFileItem[];
      total: number;
      page?: number;
      size?: number;
    };

    /** 音乐信息 */
    type MusicInfo = {
      /** 歌曲名称 */
      songName: string;
      /** 歌手 */
      singer: string;
      /** 专辑 */
      album: string;
      /** 封面 Base64 编码 */
      coverBase64?: string;
    };

    /** 视频信息 */
    type VideoInfo = {
      /** 视频宽度 */
      width?: number;
      /** 视频高度 */
      height?: number;
      /** 视频码率 */
      bitrate?: string;
      /** 视频格式 */
      format?: string;
      /** 视频帧率 */
      frameRate?: number;
      /** 视频时长 */
      duration?: string;
    };

    /** 文件项 */
    type FileItem = Common.CommonRecord<{
      /** 文件ID */
      fileId: CommonType.IdType;
      /** 文件名 */
      fileName: string;
      /** 文件类型 (folder/image/document/video/audio/other) */
      fileType: string;
      /** 文件大小 (字节) */
      fileSize: number;
      /** 文件扩展名 */
      fileExtension?: string;
      /** 父文件夹ID */
      parentId: CommonType.IdType | null;
      /** 文件路径 */
      filePath: string;
      /** 修改时间 */
      modifyTime: string;
      /** 是否为文件夹 */
      isFolder: boolean;
      /** 文件URL (非文件夹) */
      fileUrl?: string;
      /** 文件图标 (iconify 名称) */
      icon?: string;
      /** 是否有媒体封面/缩略图 */
      mediaCover?: boolean;
      /** 是否显示封面 */
      showCover?: boolean;
      /** 音乐信息 */
      music?: MusicInfo;
      /** 视频信息 */
      video?: VideoInfo;
      /** MIME类型 (后端返回) */
      contentType?: string;
      /** 是否已收藏 */
      isFavorite?: boolean;
      /** 是否已分享（外链） */
      isShare?: boolean;
      /** 已共享给用户数 */
      sharedUserCount?: number;
      /** 已共享给部门数 */
      sharedDeptCount?: number;
    }> & {
      /** 兼容属性: 文件ID别名 */
      id?: CommonType.IdType;
      /** 兼容属性: 文件名别名 */
      name?: string;
      /** 兼容属性: 文件大小别名 */
      size?: number;
      /** 兼容属性: 是否文件夹别名 */
      isDir?: boolean;
      /** 兼容属性: 扩展名别名 */
      extendName?: string;
    };

    /** 文件列表响应 */
    type FileList = Common.PaginatingQueryRecord<FileItem>;

    /** 文件搜索参数 */
    type FileSearchParams = CommonType.RecordNullable<{
      /** 文件类型筛选 */
      fileType: FileType | null;
      /** 搜索关键词 */
      keyword: string | null;
      /** 父文件夹ID */
      parentId: CommonType.IdType | null;
      /** 排序字段 */
      sortField: 'name' | 'size' | 'modifyTime' | 'type' | null;
      /** 排序方式 */
      sortOrder: 'asc' | 'desc' | null;
      /** 分页参数 */
      pageNum: number;
      pageSize: number;
      /** 是否包含挂载项数据（第1期未启用） */
      includeMounts?: boolean;
    }>;

    /** 面包屑项 */
    type BreadcrumbItem = {
      /** 文件夹ID (null表示根目录) */
      fileId: CommonType.IdType | null;
      /** 文件夹名称 */
      fileName: string;
      /** 路径字符串 */
      filePath: string;
    };

    /** 路径解析响应（/file-meta/path-resolve） */
    type PathResolveResponse = {
      /** 最终文件夹ID */
      fileId: CommonType.IdType;
      /** 文件夹名称 */
      fileName: string;
      /** 父目录ID */
      parentId: CommonType.IdType | null;
      /** 路径字符串 */
      filePath: string;
      /** 面包屑链 */
      breadcrumb: BreadcrumbItem[];
    };

    /** 配额信息 */
    type QuotaInfo = {
      /** 已用空间（字节） */
      usedSpace: number;
      /** 配额上限（字节） */
      quota: number;
      /** 是否无限制 */
      unlimited: boolean;
      /** 配额来源 */
      quotaSource: 'personal' | 'global' | 'none';
    };

    /** 文件操作类型(右键菜单/操作按钮) */
    type DiskActionType = 'rename' | 'move' | 'copy' | 'delete' | 'download';

    /** 新建文件夹请求(POST /file-meta/mkdir) */
    type MkdirParams = {
      /** 父目录全路径(根='/') */
      parentPath: string;
      /** 文件夹名 */
      folderName: string;
    };

    /** 重命名请求(POST /file-meta/rename) */
    type RenameParams = {
      /** 文件ID */
      fileId: CommonType.IdType;
      /** 新名称 */
      newName: string;
    };

    /** 新建空文件请求(POST /file-meta/create-file) */
    type CreateFileParams = {
      /** 父目录全路径(根='/') */
      parentPath: string;
      /** 文件名(含扩展名) */
      fileName: string;
    };

    /** 新建空文件结果 */
    type CreateFileResult = {
      /** 新建的 fileId */
      fileId: string;
    };

    /** 移动请求(PUT /file-meta/move,A2) */
    type MoveParams = {
      fileIds: CommonType.IdType[];
      targetPath: string;
    };

    /** 复制请求(POST /file-meta/copy,A2) */
    type CopyParams = {
      fileIds: CommonType.IdType[];
      targetPath: string;
    };

    /** 删除(移入回收站)请求(POST /file-meta/delete) */
    type DeleteParams = {
      fileIds: CommonType.IdType[];
    };

    /** 目录树节点(GET /file-meta/folder-tree,移动/复制目标选择器) */
    type FolderTreeNode = {
      id: string;
      name: string;
      path: string;
      children: FolderTreeNode[];
    };

    /** 上传检测请求(GET /file-meta/upload) */
    type UploadCheckParams = {
      identifier: string;
      quickHash: string;
      strongHash: string;
      /** 中间2MB MD5(>4MB 文件,秒传二次校验防首尾碰撞) */
      midHash: string;
      fileName: string;
      totalSize: number;
      totalChunks: number;
      chunkSize: number;
      currentDirectory: string;
      relativePath?: string;
    };

    /** 上传检测结果(秒传 pass=true / 续传 resume[]=已收分片) */
    type UploadCheckResp = {
      pass: boolean;
      fileId?: string;
      /** 秒传源文件ID(pass=true 时有值,前端 merge(instant=true) 透传复用) */
      sourceFileId?: string;
      uploadId: string;
      resume: number[];
      merge: boolean;
    };

    /** 上传分片请求(POST /file-meta/upload,multipart) */
    type UploadChunkParams = {
      uploadId: string;
      chunkNumber: number;
      chunkHash: string;
      file: Blob;
    };

    /** 合并请求(POST /file-meta/merge) */
    type UploadMergeParams = {
      identifier: string;
      fileName: string;
      totalSize: number;
      totalChunks: number;
      currentDirectory: string;
      relativePath?: string;
      quickHash: string;
      strongHash: string;
      /** 中间2MB MD5(秒传二次校验) */
      midHash: string;
      /** 秒传复用模式(Check 命中 pass=true 时前端置 true,后端不合并分片直接建引用节点) */
      instant?: boolean;
    };

    /** 合并结果 */
    type UploadMergeResp = {
      fileId: string;
      url: string;
    };

    /** 批量预建目录请求(POST /file-meta/ensure-folders,文件夹上传前预建含空目录) */
    type EnsureFoldersParams = {
      currentDirectory: string;
      /** 相对 currentDirectory 的目录路径列表(如 ['a/b','a/c'],含空目录) */
      paths: string[];
    };

    /** 上传任务状态 */
    type UploadTaskStatus = 'pending' | 'hashing' | 'uploading' | 'paused' | 'merging' | 'success' | 'error';

    /** 上传任务(传输面板) */
    type UploadTask = {
      id: string;
      name: string;
      size: number;
      progress: number; // 0-100
      status: UploadTaskStatus;
      errorMsg?: string;
      relativePath?: string;
      /** 已传字节数(详情展示用) */
      transferredSize?: number;
      /** 上传速度 bytes/s(详情展示用) */
      speed?: number;
      /** 预计剩余秒数 */
      remainingTime?: number;
      /** 所属文件夹聚合组 id(子文件有;聚合条目自身 id===folderId) */
      folderId?: string;
      /** 是否文件夹聚合条目(汇总其 folderId 下子文件) */
      isFolderAgg?: boolean;
      /** 文件夹名(聚合条目用) */
      folderName?: string;
      /** 子文件完成数(聚合条目用) */
      completedCount?: number;
      /** 子文件总数(聚合条目用) */
      totalCount?: number;
      /** 文件夹总大小(聚合条目用) */
      folderTotalSize?: number;
      /** 文件夹已传大小(聚合条目用) */
      folderTransferredSize?: number;
    };
  }
}
