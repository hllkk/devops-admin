<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue';
import { NCheckbox, type DropdownOption } from 'naive-ui';
import { useDiskStore } from '@/store/modules/disk';
import { useDiskCreate } from '@/hooks/business/disk/use-disk-create';
import { $t } from '@/locales';
import FileIcon from './file-icon.vue';
import { formatFileSize } from '@/utils/format';

defineOptions({ name: 'FileCard' });

interface Props {
  file: Api.Disk.FileItem;
}

interface Emits {
  (e: 'action', type: Api.Disk.DiskActionType, file: Api.Disk.FileItem): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();
const diskStore = useDiskStore();
const { submit, cancel, keydown, blur } = useDiskCreate();

/** 行内重命名态:当前卡片即为重命名目标 */
const isRenaming = computed(() => !!diskStore.renamingId && diskStore.renamingId === props.file.fileId);
const inputRef = ref<{ focus: () => void; select: () => void } | null>(null);
watch(isRenaming, renaming => {
  if (!renaming) return;
  nextTick(() => {
    inputRef.value?.focus();
    inputRef.value?.select();
  });
});

const selected = computed(() => diskStore.selectedFiles.includes(props.file.fileId));

/**
 * 文件夹展示创建时间(月-日 时:分),卡片底部紧凑格式。
 * 文件展示大小、文件夹展示创建时间,二者占用同一行位。
 */
const folderCreateTime = computed(() => {
  const t = props.file.createTime;
  if (!t) return '';
  const d = new Date(t);
  if (Number.isNaN(d.getTime())) return '';
  const mo = String(d.getMonth() + 1).padStart(2, '0');
  const da = String(d.getDate()).padStart(2, '0');
  const h = String(d.getHours()).padStart(2, '0');
  const mi = String(d.getMinutes()).padStart(2, '0');
  return `${mo}-${da} ${h}:${mi}`;
});

/** 图标尺寸随网格大小档位（大图 120 / 缩略 80） */
const iconSize = computed(() => (diskStore.gridSize === 'large' ? 120 : 80));

/** 左上角复选框勾选/取消：同步到 store */
function toggleSelect(checked: boolean) {
  const id = props.file.fileId;
  if (checked) {
    diskStore.setSelectedFiles([...diskStore.selectedFiles, id]);
  } else {
    diskStore.setSelectedFiles(diskStore.selectedFiles.filter(f => f !== id));
  }
}

function handleDblClick() {
  // 第1期只读：双击文件夹进入；文件预览后续期补充
  if (isRenaming.value) return;
  if (props.file.isFolder) diskStore.enterFolder(props.file);
}

/** ⋯ 操作菜单(第2期:重命名/删除;移动/复制 A2;下载仅文件) */
const opOptions = computed<DropdownOption[]>(() => {
  const opts: DropdownOption[] = [];
  // 仅文件可下载(后端拒绝目录下载);目录无此项
  if (!props.file.isFolder) {
    opts.push({ label: $t('page.disk.action.download'), key: 'download' });
  }
  opts.push(
    { label: $t('page.disk.action.rename'), key: 'rename' },
    { label: $t('page.disk.action.move'), key: 'move' },
    { label: $t('page.disk.action.copy'), key: 'copy' },
    { label: $t('page.disk.action.delete'), key: 'delete' }
  );
  return opts;
});

function handleAction(key: string) {
  emit('action', key as Api.Disk.DiskActionType, props.file);
}
</script>

<template>
  <div
    class="relative group flex-col-center cursor-pointer gap-6px p-10px rd-8px overflow-hidden transition-colors"
    :class="selected ? 'bg-primary/15 hover:bg-primary/20' : 'hover:bg-primary/10'"
    @dblclick="handleDblClick"
  >
    <!-- 左上角选中复选框:hover 浮现,选中后常驻;重命名态隐藏 -->
    <NCheckbox
      v-if="!isRenaming"
      :checked="selected"
      class="absolute left-4px top-4px z-1 transition-opacity"
      :class="selected ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'"
      @click.stop
      @update:checked="toggleSelect"
    />
    <!-- ⋯ 操作菜单(hover 显示;重命名态隐藏) -->
    <NDropdown v-if="!isRenaming" :options="opOptions" trigger="click" placement="bottom-end" @select="handleAction">
      <NButton
        quaternary
        size="tiny"
        :focusable="false"
        class="absolute right-4px top-4px opacity-0 transition-opacity group-hover:opacity-100"
        @click.stop
      >
        <SvgIcon icon="material-symbols:more-vert" class="text-16px" />
      </NButton>
    </NDropdown>
    <template v-if="isRenaming">
      <div class="flex w-full justify-center">
        <FileIcon :file-type="file.fileType" :extension="file.fileExtension" :size="iconSize" />
      </div>
      <div class="w-full px-4px" @click.stop>
        <NInput
          ref="inputRef"
          :value="diskStore.creatingName"
          size="small"
          :placeholder="$t('page.disk.modal.namePlaceholder')"
          @update:value="(v: string) => diskStore.setCreatingName(v)"
          @keydown="keydown"
          @blur="blur"
        />
      </div>
      <div class="absolute right-4px top-4px flex items-center gap-2px" @click.stop @mousedown.prevent>
        <NButton text size="tiny" :focusable="false" @click="submit">
          <SvgIcon icon="material-symbols:check" class="text-16px text-primary" />
        </NButton>
        <NButton text size="tiny" :focusable="false" @click="cancel">
          <SvgIcon icon="material-symbols:close" class="text-16px opacity-50" />
        </NButton>
      </div>
    </template>
    <template v-else>
      <FileIcon :file-type="file.fileType" :extension="file.fileExtension" :size="iconSize" />
      <span
        class="w-full line-clamp-2 break-all text-center text-13px leading-18px min-h-36px"
        :title="file.fileName"
      >{{ file.fileName }}</span>
      <span v-if="!file.isFolder" class="text-11px opacity-50">{{ formatFileSize(file.fileSize) }}</span>
      <span v-else class="text-11px opacity-50">{{ folderCreateTime }}</span>
    </template>
  </div>
</template>
