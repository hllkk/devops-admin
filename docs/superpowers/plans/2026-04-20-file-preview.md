# 文件在线预览功能 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为运维管理系统网盘模块实现完整的文件在线预览功能，支持图片、PDF、文本/代码、视频、音频、Office 文档六种文件类型的弹窗式预览。

**Architecture:** 前端基于 Vue3 + NaiveUI 的 NModal 弹窗组件，根据文件后缀路由到对应预览子组件（ImageViewer/PdfViewer/CodeViewer/VideoPlayer/AudioPlayer/OfficeViewer）。重型库按需动态 import。后端复用已有 PreviewService，新增统一预览接口设置正确 Content-Type 和 Range 支持。

**Tech Stack:** v-viewer, @embedpdf/vue-pdf-viewer, monaco-editor, artplayer, APlayer, OnlyOffice Document Server

---

## File Structure

### Frontend (新建)

| 文件 | 职责 |
|------|------|
| `src/utils/file-type.ts` | 文件类型判断工具函数 |
| `src/views/disk/modules/preview/file-preview-modal.vue` | 主入口弹窗，根据文件类型分发子组件 |
| `src/views/disk/modules/preview/image-viewer.vue` | 图片预览 (v-viewer) |
| `src/views/disk/modules/preview/pdf-viewer.vue` | PDF 预览 (@embedpdf/vue-pdf-viewer) |
| `src/views/disk/modules/preview/code-viewer.vue` | 代码/文本预览 (monaco-editor) |
| `src/views/disk/modules/preview/video-player.vue` | 视频播放 (artplayer) |
| `src/views/disk/modules/preview/audio-player.vue` | 音频播放 (APlayer) |
| `src/views/disk/modules/preview/office-viewer.vue` | Office 文档预览 (OnlyOffice iframe) |

### Frontend (修改)

| 文件 | 变更 |
|------|------|
| `src/typings/api/disk.api.d.ts` | 新增 PreviewCategory 类型 |
| `src/service/api/disk/file.ts` | 新增 `fetchPreviewUrl` 函数 |
| `src/views/disk/index.vue` | 接入 FilePreviewModal，替换 TODO |

### Backend (修改)

| 文件 | 变更 |
|------|------|
| `api/v1/disk/disk_preview.go` | 新增 `PreviewFile` 方法（统一预览接口） |
| `router/disk/disk_preview.go` | 新增 `/file/:fileId` 路由 |

---

## Task 1: Install Frontend Dependencies

**Files:**
- Modify: `frontend/package.json`

- [ ] **Step 1: Install preview libraries**

```bash
cd /home/devops-admin/frontend
pnpm add v-viewer @embedpdf/vue-pdf-viewer artplayer aplayer monaco-editor
```

- [ ] **Step 2: Verify installation**

Run: `cd /home/devops-admin/frontend && pnpm ls v-viewer @embedpdf/vue-pdf-viewer artplayer aplayer monaco-editor`
Expected: All packages listed with versions

- [ ] **Step 3: Commit**

```bash
git add frontend/package.json frontend/pnpm-lock.yaml
git commit -m "chore: 安装文件预览相关依赖"
```

---

## Task 2: Backend — Unified Preview Endpoint

**Files:**
- Modify: `backend/api/v1/disk/disk_preview.go`
- Modify: `backend/router/disk/disk_preview.go`

- [ ] **Step 1: Add PreviewFile handler to disk_preview.go**

Add the following method to `backend/api/v1/disk/disk_preview.go`:

```go
// PreviewFile
// @Tags Disk
// @Summary Preview file stream with correct Content-Type
// @Security ApiKeyAuth
// @accept application/json
// @Param fileId path string true "File ID"
// @Success 200 {file} file "File stream"
// @Router /preview/file/{fileId} [get]
func (p *PreviewApi) PreviewFile(c *gin.Context) {
	fileId := c.Param("fileId")
	if fileId == "" {
		response.FailWithMessage("fileId is required", c)
		return
	}
	userId := utils.GetUserID(c)

	file, fullPath, err := PreviewService.GetFileForPreview(fileId, userId, "")
	if err != nil {
		global.OPS_LOG.Error("PreviewFile failed", zap.Error(err))
		response.FailWithMessage("Get file failed: "+err.Error(), c)
		return
	}

	// Set correct Content-Type
	contentType := utils.GetContentType(file.Name)
	c.Header("Content-Type", contentType)

	// Set Content-Disposition for inline preview
	c.Header("Content-Disposition", "inline")

	// Support Range requests for media files
	c.Header("Accept-Ranges", "bytes")

	c.File(fullPath)
}
```

- [ ] **Step 2: Add route to disk_preview.go router**

Modify `backend/router/disk/disk_preview.go`:

```go
func (p *PreViewRouter) InitPreViewRouter(router *gin.RouterGroup) {
	previewGroup := router.Group("/preview")
	diskApi := v1.ApiV1GroupApp.DiskApiV1Group.PreviewApi
	{
		previewGroup.GET("/text/stream", diskApi.PreviewTextStream)
		previewGroup.GET("/shared/text/stream", diskApi.PreviewSharedTextStream)
		previewGroup.GET("/file/:fileId", diskApi.PreviewFile)
	}
}
```

- [ ] **Step 3: Build and verify**

Run: `cd /home/devops-admin/backend && go build ./...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add backend/api/v1/disk/disk_preview.go backend/router/disk/disk_preview.go
git commit -m "feat: 添加统一文件预览接口，支持Content-Type和Range请求"
```

---

## Task 3: Frontend — File Type Utility

**Files:**
- Create: `frontend/src/utils/file-type.ts`

- [ ] **Step 1: Create file-type.ts**

```typescript
/** 文件预览分类 */
export type PreviewCategory = 'image' | 'pdf' | 'video' | 'audio' | 'office' | 'code' | 'unknown';

/** Office 文档子类型 */
export type OfficeType = 'word' | 'excel' | 'ppt';

const IMAGE_EXTS = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg', 'ico', 'tiff', 'tif'];

const PDF_EXTS = ['pdf'];

const VIDEO_EXTS = ['mp4', 'webm', 'ogg', 'ogv', 'flv', 'avi', 'mkv', 'mov', 'wmv', 'm4v', '3gp'];

const AUDIO_EXTS = ['mp3', 'wav', 'flac', 'aac', 'ogg', 'oga', 'm4a', 'wma', 'ape', 'opus'];

const OFFICE_EXTS: Record<OfficeType, string[]> = {
  word: ['doc', 'docx'],
  excel: ['xls', 'xlsx'],
  ppt: ['ppt', 'pptx']
};

const CODE_EXTS = [
  'txt', 'log', 'json', 'xml', 'yaml', 'yml', 'toml', 'ini', 'cfg', 'conf',
  'py', 'go', 'js', 'ts', 'java', 'c', 'cpp', 'h', 'hpp', 'cs', 'sh', 'bash',
  'sql', 'md', 'markdown', 'css', 'scss', 'less', 'html', 'htm', 'vue',
  'jsx', 'tsx', 'rs', 'rb', 'php', 'swift', 'kt', 'scala', 'lua', 'pl',
  'r', 'dart', 'properties', 'env', 'gitignore', 'dockerignore', 'editorconfig'
];

/** 根据文件名获取预览分类 */
export function getPreviewCategory(filename: string): PreviewCategory {
  const ext = getExtension(filename);
  if (!ext) return 'unknown';

  const lower = ext.toLowerCase();

  if (IMAGE_EXTS.includes(lower)) return 'image';
  if (PDF_EXTS.includes(lower)) return 'pdf';
  if (VIDEO_EXTS.includes(lower)) return 'video';
  if (AUDIO_EXTS.includes(lower)) return 'audio';
  if (getAllOfficeExts().includes(lower)) return 'office';
  if (CODE_EXTS.includes(lower)) return 'code';

  return 'unknown';
}

/** 判断文件是否可预览 */
export function isPreviewable(filename: string): boolean {
  return getPreviewCategory(filename) !== 'unknown';
}

/** 获取 Office 子类型 */
export function getOfficeType(filename: string): OfficeType | null {
  const ext = getExtension(filename)?.toLowerCase();
  if (!ext) return null;

  for (const [officeType, exts] of Object.entries(OFFICE_EXTS)) {
    if (exts.includes(ext)) return officeType as OfficeType;
  }
  return null;
}

/** 获取 Monaco Editor 语言标识 */
export function getMonacoLanguage(filename: string): string {
  const ext = getExtension(filename)?.toLowerCase();
  if (!ext) return 'plaintext';

  const langMap: Record<string, string> = {
    js: 'javascript', jsx: 'javascript', ts: 'typescript', tsx: 'typescript',
    vue: 'html', html: 'html', htm: 'html', css: 'css', scss: 'scss', less: 'less',
    json: 'json', xml: 'xml', yaml: 'yaml', yml: 'yaml', toml: 'ini',
    md: 'markdown', markdown: 'markdown', sql: 'sql', py: 'python',
    go: 'go', java: 'java', c: 'c', cpp: 'cpp', h: 'c', hpp: 'cpp',
    cs: 'csharp', sh: 'shell', bash: 'shell', rs: 'rust', rb: 'ruby',
    php: 'php', swift: 'swift', kt: 'kotlin', scala: 'scala', lua: 'lua',
    pl: 'perl', r: 'r', dart: 'dart', ini: 'ini', cfg: 'ini', conf: 'ini',
    properties: 'ini', env: 'plaintext', log: 'plaintext', txt: 'plaintext'
  };

  return langMap[ext] || 'plaintext';
}

/** 获取文件扩展名（不含点） */
export function getExtension(filename: string): string | undefined {
  const parts = filename.split('.');
  if (parts.length < 2) return undefined;
  return parts[parts.length - 1];
}

function getAllOfficeExts(): string[] {
  return Object.values(OFFICE_EXTS).flat();
}
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd /home/devops-admin/frontend && pnpm typecheck`
Expected: No errors related to file-type.ts

- [ ] **Step 3: Commit**

```bash
git add frontend/src/utils/file-type.ts
git commit -m "feat: 添加文件类型判断工具函数"
```

---

## Task 4: Frontend — Preview Type & API

**Files:**
- Modify: `frontend/src/typings/api/disk.api.d.ts`
- Modify: `frontend/src/service/api/disk/file.ts`

- [ ] **Step 1: Add preview types to disk.api.d.ts**

Add inside the `Api.Disk` namespace, after the `QuotaCheckResponse` type:

```typescript
    /** 预览文件信息 */
    type PreviewFileInfo = {
      /** 文件ID */
      fileId: CommonType.IdType;
      /** 文件名 */
      fileName: string;
      /** 文件大小（字节） */
      fileSize: number;
      /** 文件扩展名 */
      fileExtension?: string;
      /** 文件路径 */
      filePath?: string;
    };
```

Note: `PreviewCategory` type is defined in `src/utils/file-type.ts` (Task 3). Do not duplicate it here.

- [ ] **Step 2: Add preview URL function to file.ts**

Add to `frontend/src/service/api/disk/file.ts`:

```typescript
/** 获取文件预览URL */
export function getPreviewUrl(fileId: CommonType.IdType): string {
  return `/disk/preview/file/${fileId}`;
}
```

Note: This constructs a URL path that maps to the backend preview endpoint via the Vite proxy.

- [ ] **Step 3: Verify**

Run: `cd /home/devops-admin/frontend && pnpm typecheck`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/typings/api/disk.api.d.ts frontend/src/service/api/disk/file.ts
git commit -m "feat: 添加预览类型定义和预览URL函数"
```

---

## Task 5: Frontend — FilePreviewModal

**Files:**
- Create: `frontend/src/views/disk/modules/preview/file-preview-modal.vue`

- [ ] **Step 1: Create FilePreviewModal component**

```vue
<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue';
import { NModal, NButton, NSpace, NTooltip } from 'naive-ui';
import { getPreviewCategory } from '@/utils/file-type';
import type { PreviewCategory } from '@/utils/file-type';
import { getPreviewUrl } from '@/service/api/disk/file';

defineOptions({ name: 'FilePreviewModal' });

interface Props {
  visible: boolean;
  file: Api.Disk.PreviewFileInfo | null;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
}>();

const ImageViewer = defineAsyncComponent(() => import('./image-viewer.vue'));
const PdfViewer = defineAsyncComponent(() => import('./pdf-viewer.vue'));
const CodeViewer = defineAsyncComponent(() => import('./code-viewer.vue'));
const VideoPlayer = defineAsyncComponent(() => import('./video-player.vue'));
const AudioPlayer = defineAsyncComponent(() => import('./audio-player.vue'));
const OfficeViewer = defineAsyncComponent(() => import('./office-viewer.vue'));

const showModal = computed({
  get: () => props.visible,
  set: val => emit('update:visible', val)
});

const previewCategory = computed<PreviewCategory>(() => {
  if (!props.file) return 'unknown';
  return getPreviewCategory(props.file.fileName);
});

const previewUrl = computed(() => {
  if (!props.file) return '';
  return getPreviewUrl(props.file.fileId);
});

const fileSize = computed(() => {
  if (!props.file) return '';
  const size = props.file.fileSize;
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  if (size < 1024 * 1024 * 1024) return `${(size / (1024 * 1024)).toFixed(1)} MB`;
  return `${(size / (1024 * 1024 * 1024)).toFixed(1)} GB`;
});

function handleDownload() {
  if (!props.file) return;
  const link = document.createElement('a');
  link.href = previewUrl.value;
  link.download = props.file.fileName;
  link.click();
}
</script>

<template>
  <NModal
    v-model:show="showModal"
    preset="card"
    :bordered="false"
    :style="{ width: '80vw', maxWidth: '1200px', height: '80vh' }"
    :mask-closable="true"
    :title="file?.fileName || '文件预览'"
  >
    <template #header-extra>
      <NSpace align="center" :size="12">
        <span v-if="file" class="text-12px text-gray-400">{{ fileSize }}</span>
        <NTooltip trigger="hover">
          <template #trigger>
            <NButton quaternary size="small" @click="handleDownload">
              <template #icon>
                <icon-ic-round-download />
              </template>
            </NButton>
          </template>
          下载文件
        </NTooltip>
      </NSpace>
    </template>

    <div class="h-full overflow-hidden">
      <template v-if="!file">
        <div class="flex items-center justify-center h-full text-gray-400">未选择文件</div>
      </template>
      <template v-else-if="previewCategory === 'image'">
        <ImageViewer :url="previewUrl" :file-name="file.fileName" />
      </template>
      <template v-else-if="previewCategory === 'pdf'">
        <PdfViewer :url="previewUrl" :file-name="file.fileName" />
      </template>
      <template v-else-if="previewCategory === 'code'">
        <CodeViewer :url="previewUrl" :file-name="file.fileName" />
      </template>
      <template v-else-if="previewCategory === 'video'">
        <VideoPlayer :url="previewUrl" :file-name="file.fileName" />
      </template>
      <template v-else-if="previewCategory === 'audio'">
        <AudioPlayer :url="previewUrl" :file-name="file.fileName" />
      </template>
      <template v-else-if="previewCategory === 'office'">
        <OfficeViewer :file="file" />
      </template>
      <template v-else>
        <div class="flex flex-col items-center justify-center h-full gap-4">
          <icon-carbon-document-unknow class="text-48px text-gray-300" />
          <span class="text-gray-400">不支持预览此类型文件</span>
          <NButton @click="handleDownload">下载文件</NButton>
        </div>
      </template>
    </div>
  </NModal>
</template>

<style scoped>
.h-full {
  height: calc(80vh - 60px);
}
</style>
```

Note: The icon names `icon-ic-round-download` and `icon-carbon-document-unknow` follow the project's existing icon naming convention using @iconify/vue with UnoCSS presets. Verify these icon names are available in the project's icon set; if not, adjust to available icons (e.g., `material-symbols:download`, `material-symbols:description`).

- [ ] **Step 2: Verify**

Run: `cd /home/devops-admin/frontend && pnpm typecheck`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/disk/modules/preview/file-preview-modal.vue
git commit -m "feat: 添加文件预览弹窗主入口组件"
```

---

## Task 6: Frontend — ImageViewer

**Files:**
- Create: `frontend/src/views/disk/modules/preview/image-viewer.vue`

- [ ] **Step 1: Create ImageViewer component**

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import Viewer from 'viewerjs';
import 'viewerjs/dist/viewer.css';

defineOptions({ name: 'ImageViewer' });

interface Props {
  url: string;
  fileName: string;
}

const props = defineProps<Props>();

const containerRef = ref<HTMLDivElement>();
let viewerInstance: Viewer | null = null;

onMounted(() => {
  if (containerRef.value) {
    viewerInstance = new Viewer(containerRef.value, {
      inline: true,
      title: false,
      toolbar: {
        zoomIn: true,
        zoomOut: true,
        oneToOne: true,
        reset: true,
        prev: false,
        play: false,
        next: false,
        rotateLeft: true,
        rotateRight: true,
        flipHorizontal: true,
        flipVertical: true
      },
      viewed() {
        // Auto-fit after image loads
      }
    });
  }
});

onUnmounted(() => {
  viewerInstance?.destroy();
  viewerInstance = null;
});
</script>

<template>
  <div class="image-viewer-container">
    <div ref="containerRef">
      <img :src="url" :alt="fileName" style="display: none" />
    </div>
  </div>
</template>

<style scoped>
.image-viewer-container {
  width: 100%;
  height: 100%;
}

.image-viewer-container :deep(.viewer-container) {
  background-color: var(--n-color-modal, #fff);
}
</style>
```

Note: viewerjs inline mode requires the container to have explicit dimensions. The parent provides `height: calc(80vh - 60px)` from the modal. If inline mode has issues, switch to imperative API: `viewerInstance = new Viewer(imgElement, { ... })` then call `viewerInstance.show()`.

- [ ] **Step 2: Verify rendering in browser**

Run: `cd /home/devops-admin/frontend && pnpm dev`
Manual: Navigate to disk page, double-click an image file. Verify the modal opens and shows the image viewer with zoom/rotate controls.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/disk/modules/preview/image-viewer.vue
git commit -m "feat: 添加图片预览组件(v-viewer/viewerjs)"
```

---

## Task 7: Frontend — Integrate Preview into Disk Page

**Files:**
- Modify: `frontend/src/views/disk/index.vue`

- [ ] **Step 1: Import FilePreviewModal**

Add to the imports section of `src/views/disk/index.vue`:

```typescript
import FilePreviewModal from './modules/preview/file-preview-modal.vue';
```

- [ ] **Step 2: Add preview state**

Add after the `renamingFile` ref (around line 30):

```typescript
// 文件预览
const previewVisible = ref(false);
const previewFile = ref<Api.Disk.PreviewFileInfo | null>(null);
```

- [ ] **Step 3: Replace handleFileDblClick**

Replace the existing `handleFileDblClick` function (around line 316-321):

```typescript
function handleFileDblClick(file: Api.Disk.FileItem) {
  if (!file.isFolder) {
    previewFile.value = {
      fileId: file.fileId,
      fileName: file.fileName,
      fileSize: file.fileSize,
      fileExtension: file.fileExtension,
      filePath: file.filePath
    };
    previewVisible.value = true;
  }
}
```

- [ ] **Step 4: Add modal to template**

Add before the closing `</template>` tag in the disk page:

```vue
<FilePreviewModal
  v-model:visible="previewVisible"
  :file="previewFile"
/>
```

- [ ] **Step 5: Verify**

Run: `cd /home/devops-admin/frontend && pnpm typecheck && pnpm lint`
Expected: No errors

Manual: Double-click an image file, verify modal opens. Double-click non-image, verify "不支持预览" placeholder shows.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/disk/index.vue
git commit -m "feat: 集成文件预览弹窗到网盘页面"
```

---

## Task 8: Frontend — PdfViewer

**Files:**
- Create: `frontend/src/views/disk/modules/preview/pdf-viewer.vue`

- [ ] **Step 1: Create PdfViewer component**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { VuePDFViewing, usePDF } from '@embedpdf/vue-pdf-viewer';

defineOptions({ name: 'PdfViewer' });

interface Props {
  url: string;
  fileName: string;
}

const props = defineProps<Props>();

const { pdf, pages, loading, error } = usePDF(props.url);
const currentPage = ref(1);
const scale = ref(1);
</script>

<template>
  <div class="pdf-viewer-container">
    <div v-if="loading" class="flex items-center justify-center h-full">
      <n-spin size="large" />
    </div>
    <div v-else-if="error" class="flex items-center justify-center h-full text-red-400">
      PDF 加载失败: {{ error }}
    </div>
    <template v-else>
      <div class="pdf-toolbar">
        <NSpace align="center" :size="8">
          <NButton size="tiny" quaternary :disabled="currentPage <= 1" @click="currentPage--">
            <icon-material-symbols-arrow-back />
          </NButton>
          <span class="text-12px">{{ currentPage }} / {{ pages }}</span>
          <NButton size="tiny" quaternary :disabled="currentPage >= pages" @click="currentPage++">
            <icon-material-symbols-arrow-forward />
          </NButton>
          <NDivider vertical />
          <NButton size="tiny" quaternary @click="scale = Math.max(0.5, scale - 0.25)">-</NButton>
          <span class="text-12px">{{ Math.round(scale * 100) }}%</span>
          <NButton size="tiny" quaternary @click="scale = Math.min(3, scale + 0.25)">+</NButton>
          <NButton size="tiny" quaternary @click="scale = 1">重置</NButton>
        </NSpace>
      </div>
      <div class="pdf-content">
        <VuePDFViewing :pdf="pdf" :page="currentPage" :scale="scale" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.pdf-viewer-container {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.pdf-toolbar {
  padding: 8px 12px;
  border-bottom: 1px solid var(--n-border-color, #e0e0e0);
  flex-shrink: 0;
}

.pdf-content {
  flex: 1;
  overflow: auto;
  display: flex;
  justify-content: center;
  padding: 16px;
}
</style>
```

Note: The `@embedpdf/vue-pdf-viewer` API may differ. Check the library's documentation for exact component names and props. If the library provides different exports, adjust accordingly. Common alternatives: `@vue-pdf-viewer/viewer`, `vue-pdf-embed`.

- [ ] **Step 2: Verify**

Run: `cd /home/devops-admin/frontend && pnpm typecheck`
Expected: No errors

Manual: Double-click a PDF file, verify it renders with page navigation and zoom.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/disk/modules/preview/pdf-viewer.vue
git commit -m "feat: 添加PDF预览组件(@embedpdf/vue-pdf-viewer)"
```

---

## Task 9: Frontend — CodeViewer

**Files:**
- Create: `frontend/src/views/disk/modules/preview/code-viewer.vue`

- [ ] **Step 1: Create CodeViewer component**

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { getMonacoLanguage } from '@/utils/file-type';

defineOptions({ name: 'CodeViewer' });

interface Props {
  url: string;
  fileName: string;
}

const props = defineProps<Props>();

const containerRef = ref<HTMLDivElement>();
const loading = ref(true);
const errorMsg = ref('');
const content = ref('');
let editor: any = null;

const language = computed(() => getMonacoLanguage(props.fileName));

const MAX_FILE_SIZE = 1 * 1024 * 1024; // 1MB

onMounted(async () => {
  try {
    const response = await fetch(props.url);
    if (!response.ok) {
      errorMsg.value = `加载失败: ${response.statusText}`;
      loading.value = false;
      return;
    }

    const text = await response.text();
    if (text.length > MAX_FILE_SIZE) {
      content.value = text.slice(0, MAX_FILE_SIZE) + '\n\n... 文件过大，仅显示前 1MB ...';
    } else {
      content.value = text;
    }

    // Dynamic import monaco-editor
    const monaco = await import('monaco-editor');

    if (containerRef.value) {
      // Determine theme from system preference
      const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;

      editor = monaco.editor.create(containerRef.value, {
        value: content.value,
        language: language.value,
        theme: isDark ? 'vs-dark' : 'vs',
        readOnly: true,
        minimap: { enabled: content.value.length < 100000 },
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        lineNumbers: 'on',
        automaticLayout: true,
        fontSize: 13,
        renderLineHighlight: 'none'
      });
    }
  } catch (err) {
    errorMsg.value = `加载失败: ${err instanceof Error ? err.message : '未知错误'}`;
  } finally {
    loading.value = false;
  }
});

onUnmounted(() => {
  editor?.dispose();
  editor = null;
});
</script>

<template>
  <div class="code-viewer-container">
    <div v-if="loading" class="flex items-center justify-center h-full">
      <n-spin size="large" />
    </div>
    <div v-else-if="errorMsg" class="flex items-center justify-center h-full text-red-400">
      {{ errorMsg }}
    </div>
    <div v-else ref="containerRef" class="code-editor" />
  </div>
</template>

<style scoped>
.code-viewer-container {
  width: 100%;
  height: 100%;
}

.code-editor {
  width: 100%;
  height: 100%;
}
</style>
```

- [ ] **Step 2: Verify**

Run: `cd /home/devops-admin/frontend && pnpm typecheck`
Expected: No errors

Manual: Double-click a .txt or .json file, verify syntax highlighting and read-only mode.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/disk/modules/preview/code-viewer.vue
git commit -m "feat: 添加代码/文本预览组件(monaco-editor)"
```

---

## Task 10: Frontend — VideoPlayer

**Files:**
- Create: `frontend/src/views/disk/modules/preview/video-player.vue`

- [ ] **Step 1: Create VideoPlayer component**

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import Artplayer from 'artplayer';

defineOptions({ name: 'VideoPlayer' });

interface Props {
  url: string;
  fileName: string;
}

const props = defineProps<Props>();

const containerRef = ref<HTMLDivElement>();
let artInstance: Artplayer | null = null;

onMounted(() => {
  if (!containerRef.value) return;

  artInstance = new Artplayer({
    container: containerRef.value,
    url: props.url,
    title: props.fileName,
    volume: 0.8,
    autoplay: false,
    pip: true,
    fullscreen: true,
    fullscreenWeb: true,
    miniProgressBar: true,
    theme: '#1890ff',
    lang: 'zh-cn'
  });
});

onUnmounted(() => {
  artInstance?.destroy(false);
  artInstance = null;
});
</script>

<template>
  <div class="video-player-container">
    <div ref="containerRef" class="artplayer-app" />
  </div>
</template>

<style scoped>
.video-player-container {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #000;
}

.artplayer-app {
  width: 100%;
  height: 100%;
}
</style>
```

- [ ] **Step 2: Verify**

Run: `cd /home/devops-admin/frontend && pnpm typecheck`
Expected: No errors

Manual: Double-click an .mp4 file, verify video plays with controls.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/disk/modules/preview/video-player.vue
git commit -m "feat: 添加视频播放组件(artplayer)"
```

---

## Task 11: Frontend — AudioPlayer

**Files:**
- Create: `frontend/src/views/disk/modules/preview/audio-player.vue`

- [ ] **Step 1: Create AudioPlayer component**

```vue
<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import APlayer from 'aplayer';
import 'aplayer/dist/APlayer.min.css';

defineOptions({ name: 'AudioPlayer' });

interface Props {
  url: string;
  fileName: string;
}

const props = defineProps<Props>();

const containerRef = ref<HTMLDivElement>();
let playerInstance: APlayer | null = null;

onMounted(() => {
  if (!containerRef.value) return;

  // Strip extension for display name
  const nameWithoutExt = props.fileName.replace(/\.[^.]+$/, '');

  playerInstance = new APlayer({
    container: containerRef.value,
    audio: [{
      name: nameWithoutExt,
      url: props.url,
      theme: '#1890ff'
    }],
    autoplay: false,
    theme: '#1890ff',
    loop: 'none',
    order: 'list',
    preload: 'metadata',
    volume: 0.7,
    mutex: true,
    listFolded: true,
    listMaxHeight: 0
  });
});

onUnmounted(() => {
  playerInstance?.destroy();
  playerInstance = null;
});
</script>

<template>
  <div class="audio-player-container">
    <div class="audio-inner">
      <div class="audio-icon">
        <icon-material-symbols-audiotrack class="text-64px text-primary" />
      </div>
      <div class="audio-player-wrap">
        <div ref="containerRef" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.audio-player-container {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.audio-inner {
  width: 100%;
  max-width: 480px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
}

.audio-icon {
  opacity: 0.6;
}

.audio-player-wrap {
  width: 100%;
}
</style>
```

- [ ] **Step 2: Verify**

Run: `cd /home/devops-admin/frontend && pnpm typecheck`
Expected: No errors

Manual: Double-click an .mp3 file, verify audio plays with APlayer UI.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/disk/modules/preview/audio-player.vue
git commit -m "feat: 添加音频播放组件(APlayer)"
```

---

## Task 12: Frontend — OfficeViewer

**Files:**
- Create: `frontend/src/views/disk/modules/preview/office-viewer.vue`

- [ ] **Step 1: Create OfficeViewer component**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { request } from '@/service/request';

defineOptions({ name: 'OfficeViewer' });

interface Props {
  file: Api.Disk.PreviewFileInfo;
}

const props = defineProps<Props>();

const officeUrl = ref('');
const loading = ref(true);
const errorMsg = ref('');

onMounted(async () => {
  try {
    // Get OnlyOffice config from backend
    const { data } = await request<any>({
      url: '/disk/office/config',
      method: 'get',
      params: {
        fileId: props.file.fileId,
        fileName: props.file.fileName,
        mode: 'view'
      }
    });

    if (data) {
      // Build OnlyOffice editor URL
      const docServerUrl = import.meta.env.VITE_OFFICE_URL || 'http://localhost:8080';
      officeUrl.value = `${docServerUrl}/web-apps/apps/documenteditor/index.html`;
      // The actual integration uses OnlyOffice JS API with the config
    }
  } catch (err) {
    errorMsg.value = 'OnlyOffice 服务不可用，请确认服务已启动';
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div class="office-viewer-container">
    <div v-if="loading" class="flex items-center justify-center h-full">
      <n-spin size="large" />
    </div>
    <div v-else-if="errorMsg" class="flex items-center justify-center h-full text-red-400">
      {{ errorMsg }}
    </div>
    <iframe
      v-else
      :src="officeUrl"
      class="office-iframe"
      allowfullscreen
    />
  </div>
</template>

<style scoped>
.office-viewer-container {
  width: 100%;
  height: 100%;
}

.office-iframe {
  width: 100%;
  height: 100%;
  border: none;
}
</style>
```

Note: This is a simplified iframe-based approach. The full OnlyOffice integration requires loading the OnlyOffice JS API (`api.js`) and calling `new DocsAPI.DocEditor("placeholder", config)`. The backend already has the config and JWT endpoints. For a complete integration, replace the iframe with:

```html
<div id="office-editor-placeholder" style="width:100%;height:100%"></div>
```

And in `onMounted`:
```typescript
const docServerUrl = import.meta.env.VITE_OFFICE_URL || 'http://localhost:8080';
const script = document.createElement('script');
script.src = `${docServerUrl}/web-apps/apps/api/documents/api.js`;
script.onload = () => {
  new (window as any).DocsAPI.DocEditor('office-editor-placeholder', config);
};
document.head.appendChild(script);
```

Use the full DocsAPI approach if OnlyOffice Document Server is deployed and the backend config endpoint returns the correct document config.

- [ ] **Step 2: Verify**

Run: `cd /home/devops-admin/frontend && pnpm typecheck`
Expected: No errors

Manual: Double-click a .docx file. If OnlyOffice server is running, verify the document loads. If not running, verify the error message displays.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/disk/modules/preview/office-viewer.vue
git commit -m "feat: 添加Office文档预览组件(OnlyOffice)"
```

---

## Task 13: Final Integration & Verification

**Files:**
- All previously created files

- [ ] **Step 1: Run full typecheck**

Run: `cd /home/devops-admin/frontend && pnpm typecheck`
Expected: 0 errors

- [ ] **Step 2: Run full lint**

Run: `cd /home/devops-admin/frontend && pnpm lint`
Expected: 0 errors (or fix any lint issues)

- [ ] **Step 3: Verify all file types**

Manual: Test each file type:
1. Image (.jpg/.png) — verify viewerjs inline viewer with zoom/rotate
2. PDF (.pdf) — verify page navigation and zoom
3. Code (.txt/.json/.go) — verify syntax highlighting and read-only
4. Video (.mp4) — verify playback with controls
5. Audio (.mp3) — verify APlayer playback
6. Office (.docx) — verify OnlyOffice iframe or error if server not running
7. Unknown (.xyz) — verify "不支持预览" fallback

- [ ] **Step 4: Backend build verification**

Run: `cd /home/devops-admin/backend && go build ./...`
Expected: No errors

- [ ] **Step 5: Delete backend build artifacts**

Run: `cd /home/devops-admin/backend && rm -f devopsAdmin`
Expected: Build artifacts removed

- [ ] **Step 6: Final commit (if any fixes)**

```bash
git add -A
git commit -m "fix: 文件预览功能集成修复"
```
