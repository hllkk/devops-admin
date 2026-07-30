<script setup lang="ts">
import { computed, ref } from 'vue';

import apkIcon from '@/assets/imgs/file-icons/apk.png';
import audioIcon from '@/assets/imgs/file-icons/audio.png';
import btIcon from '@/assets/imgs/file-icons/bt.png';
import codeIcon from '@/assets/imgs/file-icons/code.png';
import defaultIcon from '@/assets/imgs/file-icons/default.png';
import dmgIcon from '@/assets/imgs/file-icons/dmg.png';
import docIcon from '@/assets/imgs/file-icons/doc.png';
import exeIcon from '@/assets/imgs/file-icons/exe.png';
import flvIcon from '@/assets/imgs/file-icons/flv.png';
import folderIcon from '@/assets/imgs/file-icons/folder.png';
import imageIcon from '@/assets/imgs/file-icons/image.png';
import pdfIcon from '@/assets/imgs/file-icons/pdf.png';
import pptIcon from '@/assets/imgs/file-icons/ppt.png';
import ttfIcon from '@/assets/imgs/file-icons/ttf.png';
import txtIcon from '@/assets/imgs/file-icons/txt.png';
import videoIcon from '@/assets/imgs/file-icons/video.png';
import xlsIcon from '@/assets/imgs/file-icons/xls.png';
import zipIcon from '@/assets/imgs/file-icons/zip.png';
import { getServiceBaseURL } from '@/utils/service';

defineOptions({ name: 'FileIcon' });

interface Props {
  /** 文件类型 (folder/image/document/video/audio/other) */
  fileType: string;
  /** 文件扩展名（不含点号） */
  extension?: string;
  /** small/medium/large 预设尺寸，也支持直接传 px 数值 */
  size?: 'small' | 'medium' | 'large' | number;
  /** 文件 ID，用于请求缩略图/封面 */
  fileId?: CommonType.IdType;
  /** 是否展示媒体缩略图/封面（需同时提供 fileId） */
  mediaCover?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  extension: undefined,
  size: 'medium',
  fileId: undefined,
  mediaCover: false
});

/** 扩展名 → 图标 */
const extensionIconMap: Record<string, string> = {
  pdf: pdfIcon,
  doc: docIcon,
  docx: docIcon,
  xls: xlsIcon,
  xlsx: xlsIcon,
  csv: xlsIcon,
  ppt: pptIcon,
  pptx: pptIcon,
  txt: txtIcon,
  md: txtIcon,
  log: txtIcon,
  jpg: imageIcon,
  jpeg: imageIcon,
  png: imageIcon,
  gif: imageIcon,
  webp: imageIcon,
  svg: imageIcon,
  bmp: imageIcon,
  ico: imageIcon,
  mp4: videoIcon,
  avi: videoIcon,
  mov: videoIcon,
  mkv: videoIcon,
  wmv: videoIcon,
  flv: flvIcon,
  mp3: audioIcon,
  wav: audioIcon,
  flac: audioIcon,
  aac: audioIcon,
  ogg: audioIcon,
  wma: audioIcon,
  ape: audioIcon,
  zip: zipIcon,
  rar: zipIcon,
  '7z': zipIcon,
  tar: zipIcon,
  gz: zipIcon,
  json: codeIcon,
  js: codeIcon,
  ts: codeIcon,
  vue: codeIcon,
  html: codeIcon,
  css: codeIcon,
  py: codeIcon,
  java: codeIcon,
  go: codeIcon,
  xml: codeIcon,
  yaml: codeIcon,
  yml: codeIcon,
  sql: codeIcon,
  apk: apkIcon,
  exe: exeIcon,
  msi: exeIcon,
  dmg: dmgIcon,
  ttf: ttfIcon,
  otf: ttfIcon,
  woff: ttfIcon,
  woff2: ttfIcon,
  torrent: btIcon
};

/** 文件类型 → 图标 */
const fileTypeIconMap: Record<string, string> = {
  folder: folderIcon,
  image: imageIcon,
  document: docIcon,
  video: videoIcon,
  audio: audioIcon,
  other: defaultIcon
};

/** 预设尺寸 → px */
const iconSizeMap: Record<string, number> = {
  small: 36,
  medium: 40,
  large: 80
};

const thumbnailError = ref(false);

const icon = computed(() => {
  if (props.fileType === 'folder') return folderIcon;
  if (props.extension) {
    // 处理扩展名：去掉前导的点号（如 '.md' -> 'md'）
    const ext = props.extension.toLowerCase().replace(/^\./, '');
    if (extensionIconMap[ext]) return extensionIconMap[ext];
  }
  return fileTypeIconMap[props.fileType] || defaultIcon;
});

/** 兼容枚举预设与直接 px 数值 */
const iconSize = computed(() => (typeof props.size === 'number' ? props.size : iconSizeMap[props.size]));

const isVideo = computed(() => props.fileType === 'video' || props.fileType === 'audio');

const thumbnailUrl = computed(() => {
  if (!props.mediaCover || !props.fileId || props.fileType === 'folder') return '';
  if (thumbnailError.value) return '';
  const isHttpProxy = import.meta.env.DEV && import.meta.env.VITE_HTTP_PROXY === 'Y';
  const { baseURL } = getServiceBaseURL(import.meta.env, isHttpProxy);
  const separator = isVideo.value ? '/view/cover' : '/view/thumbnail';
  return `${baseURL}${separator}?id=${props.fileId}`;
});

const showThumbnail = computed(() => thumbnailUrl.value !== '');
const showPlayIcon = computed(() => showThumbnail.value && props.fileType === 'video');

/** size 为数值时按 ≤36 判定 small（决定播放按钮尺寸） */
const isSmall = computed(() => (typeof props.size === 'number' ? props.size <= 36 : props.size === 'small'));
const playButtonSize = computed(() => (isSmall.value ? 'w-4 h-4' : 'w-7 h-7'));
const playIconSize = computed(() => (isSmall.value ? 12 : 20));

function handleThumbnailError() {
  thumbnailError.value = true;
}
</script>

<template>
  <!-- 缩略图容器：固定尺寸 + 居中裁剪 -->
  <div
    v-if="showThumbnail"
    class="shrink-0 select-none rd-4px relative overflow-hidden"
    :style="{ width: `${iconSize}px`, height: `${iconSize}px` }"
  >
    <img
      :src="thumbnailUrl"
      class="w-full h-full object-cover object-center"
      draggable="false"
      @error="handleThumbnailError"
    />
    <!-- 视频播放图标 -->
    <div v-if="showPlayIcon" class="absolute inset-0 flex items-center justify-center">
      <div class="rd-full flex items-center justify-center bg-black/40 backdrop-blur-sm" :class="[playButtonSize]">
        <SvgIcon icon="material-symbols:play-arrow" class="text-white" :style="{ fontSize: `${playIconSize}px` }" />
      </div>
    </div>
  </div>
  <!-- 静态图标 -->
  <img v-else :src="icon" :width="iconSize" :height="iconSize" class="shrink-0 select-none" draggable="false" />
</template>
