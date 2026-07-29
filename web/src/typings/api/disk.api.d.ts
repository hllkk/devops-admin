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
  }
}
