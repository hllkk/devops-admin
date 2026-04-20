# 文件在线预览功能设计

## 概述

为运维管理系统的网盘模块实现完整的文件在线预览功能，支持图片、PDF、文本/代码、视频、音频、Office 文档等常见文件类型的弹窗式在线预览。

## 技术选型

| 文件类型 | 组件库 | 加载方式 |
|---------|--------|---------|
| 图片 | `v-viewer` (viewerjs Vue3 版) | 按需动态 import |
| PDF | `@embedpdf/vue-pdf-viewer` | 按需动态 import |
| 文本/代码 | `monaco-editor` | 按需动态 import |
| 视频 | `artplayer` | 按需动态 import |
| 音频 | `APlayer` (vue3-aplayer) | 按需动态 import |
| Office | OnlyOffice Document Server | iframe 嵌入（独立 Docker 服务） |

## 架构设计

```
FilePreviewModal (主入口 - NModal)
├── FilePreviewRouter (根据文件类型分发)
│   ├── ImageViewer    (.jpg/.png/.gif/.webp/.svg/.bmp/.ico/.tiff)
│   ├── PdfViewer      (.pdf)
│   ├── CodeViewer     (.txt/.log/.json/.xml/.yaml/.py/.go/.sh/...)
│   ├── VideoPlayer    (.mp4/.webm/.ogg/.flv/.avi/.mkv/.mov/.wmv)
│   ├── AudioPlayer    (.mp3/.wav/.flac/.aac/.ogg/.m4a/.wma/.ape)
│   └── OfficeViewer   (.doc/.docx/.xls/.xlsx/.ppt/.pptx)
```

### 核心设计原则

- `FilePreviewModal` 作为唯一的弹窗入口，接收文件信息 prop
- 内部根据文件后缀路由到对应预览组件
- 所有重型库按需动态 import，不影响首屏加载
- 后端提供统一的文件预览流接口

## 组件详细设计

### FilePreviewModal（主入口）

- 基于 NaiveUI `NModal`，默认宽度 80vw / 高度 80vh
- Props: `visible` (boolean), `file` ({ id, name, size, type, url })
- 顶部工具栏：文件名、文件大小、下载按钮、全屏切换、关闭
- 全屏模式通过 NModal 的 `transform-origin` 实现
- Emits: `update:visible`, `download`

### ImageViewer

- 使用 `v-viewer` 的组件模式
- 支持缩放（滚轮/按钮）、旋转、翻转
- 多图场景支持左右切换
- 底部缩略图导航条

### PdfViewer

- 使用 `@embedpdf/vue-pdf-viewer`
- 工具栏：翻页（上/下）、页码跳转、缩放（放大/缩小/适应宽度）
- 支持侧边目录导航（如有书签信息）

### CodeViewer

- 使用 `monaco-editor` 只读模式（`readOnly: true`）
- 根据文件后缀自动设置语言模式
- 复用后端已有 `/preview/text/stream` 接口获取内容
- 主题跟随系统（dark/light）
- 大文件处理：限制加载前 1MB 内容，超过提示文件过大

### VideoPlayer

- 使用 `artplayer`
- 支持 mp4/webm 原生格式
- 通过 artplayer 插件支持 flv (flv.js) 和 hls (hls.js)
- 控件：播放/暂停、进度条、音量、全屏、倍速

### AudioPlayer

- 使用 `APlayer`（vue3-aplayer 封装）
- 显示文件名作为标题
- 封面使用后端已有的 cover 接口
- 控件：播放/暂停、进度条、音量、循环模式

### OfficeViewer

- 使用 OnlyOffice Document Server
- iframe 嵌入 OnlyOffice 编辑器
- 配置为只读预览模式（`mode: "view"`）
- 后端需实现 OnlyOffice 回调接口（保存/状态）
- OnlyOffice URL 通过环境变量 `VITE_OFFICE_URL` 配置

## 文件类型判断

### 类型映射

```typescript
const FILE_CATEGORIES = {
  image: ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg', 'ico', 'tiff', 'tif'],
  pdf: ['pdf'],
  video: ['mp4', 'webm', 'ogg', 'ogv', 'flv', 'avi', 'mkv', 'mov', 'wmv', 'm4v', '3gp'],
  audio: ['mp3', 'wav', 'flac', 'aac', 'ogg', 'oga', 'm4a', 'wma', 'ape', 'opus'],
  office: {
    word: ['doc', 'docx'],
    excel: ['xls', 'xlsx'],
    ppt: ['ppt', 'pptx'],
  },
  code: [
    'txt', 'log', 'json', 'xml', 'yaml', 'yml', 'toml', 'ini', 'cfg', 'conf',
    'py', 'go', 'js', 'ts', 'java', 'c', 'cpp', 'h', 'hpp', 'cs', 'sh', 'bash',
    'sql', 'md', 'markdown', 'css', 'scss', 'less', 'html', 'htm', 'vue',
    'jsx', 'tsx', 'rs', 'rb', 'php', 'swift', 'kt', 'scala', 'lua', 'pl',
    'r', 'dart', 'dockerfile', 'makefile', 'gitignore', 'env',
  ],
}
```

### 工具函数

```typescript
// src/utils/file-type.ts
type FileCategory = 'image' | 'pdf' | 'video' | 'audio' | 'office' | 'code' | 'unknown'

function getFileCategory(filename: string): FileCategory
function isPreviewable(filename: string): boolean
function getMonacoLanguage(filename: string): string
```

## 后端接口

### 文件预览接口

```
GET /api/v1/disk/preview/:fileId
```

- 复用已有 `PreviewService.GetFileForPreview()` 获取文件
- 设置正确的 `Content-Type` 响应头
- 设置 `Content-Disposition: inline`（预览）或 `attachment`（下载）
- 支持 Range 请求（视频/音频拖动进度条）
- 权限校验：验证用户对文件的访问权限

### OnlyOffice 回调接口

```
POST /api/v1/disk/office/callback
```

- 接收 OnlyOffice 文档保存回调
- 验证 JWT token（可选）
- 处理文档保存状态（status: 0-7）

### 缩略图接口（已有）

```
GET /api/v1/view/thumbnail/:fileId
```

## 前端目录结构

```
src/
├── components/
│   └── preview/
│       ├── index.ts              # 统一导出
│       ├── FilePreviewModal.vue  # 主入口弹窗
│       ├── ImageViewer.vue       # 图片预览
│       ├── PdfViewer.vue         # PDF 预览
│       ├── CodeViewer.vue        # 代码/文本预览
│       ├── VideoPlayer.vue       # 视频播放
│       ├── AudioPlayer.vue       # 音频播放
│       └── OfficeViewer.vue      # Office 文档预览
├── utils/
│   └── file-type.ts              # 文件类型判断工具
└── hooks/
    └── use-file-preview.ts       # 预览逻辑 hook
```

## 交互流程

1. 用户在文件列表中双击/右键"预览"文件
2. 前端调用 `useFilePreview` hook 的 `openPreview(file)` 方法
3. `FilePreviewModal` 弹窗打开，根据文件类型渲染对应组件
4. 预览组件通过 `/api/v1/disk/preview/:fileId` 获取文件流
5. 用户可在弹窗内切换全屏、下载文件、关闭预览

## 依赖安装

```bash
pnpm add v-viewer @embedpdf/vue-pdf-viewer artplayer aplayer
# monaco-editor 通过动态 import 按需加载，需要单独安装
pnpm add monaco-editor
```

## 环境变量

```
# OnlyOffice 服务地址
VITE_OFFICE_URL=http://localhost:8080
```

## 实现优先级

1. **P0 - 核心框架**: FilePreviewModal + 文件类型判断 + 图片预览
2. **P1 - 基础预览**: PDF + 代码/文本
3. **P2 - 媒体播放**: 视频 + 音频
4. **P3 - Office 预览**: OnlyOffice 集成（需部署 Document Server）
