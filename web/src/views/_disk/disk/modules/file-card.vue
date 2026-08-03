<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue';
import type { DropdownOption } from 'naive-ui';
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

/** 图标尺寸随网格大小档位（大图 80 / 小图 56） */
const iconSize = computed(() => (diskStore.gridSize === 'large' ? 80 : 56));

function handleClick() {
  const id = props.file.fileId;
  if (selected.value) {
    diskStore.setSelectedFiles(diskStore.selectedFiles.filter(f => f !== id));
  } else {
    diskStore.setSelectedFiles([...diskStore.selectedFiles, id]);
  }
}

function handleDblClick() {
  // 第1期只读：双击文件夹进入；文件预览后续期补充
  if (isRenaming.value) return;
  if (props.file.isFolder) diskStore.enterFolder(props.file);
}

/** ⋯ 操作菜单(第2期:重命名/删除;移动/复制 A2) */
const opOptions = computed<DropdownOption[]>(() => [
  { label: $t('page.disk.action.rename'), key: 'rename' },
  { label: $t('page.disk.action.move'), key: 'move' },
  { label: $t('page.disk.action.copy'), key: 'copy' },
  { label: $t('page.disk.action.delete'), key: 'delete' }
]);

function handleAction(key: string) {
  emit('action', key as Api.Disk.DiskActionType, props.file);
}
</script>

<template>
  <div
    class="relative group flex-col-center cursor-pointer gap-6px p-10px rd-8px transition-colors"
    :class="selected ? 'disk-card-active' : 'hover:bg-layout'"
    @click="handleClick"
    @dblclick="handleDblClick"
  >
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
      <span class="w-full truncate text-center text-13px" :title="file.fileName">{{ file.fileName }}</span>
      <span v-if="!file.isFolder" class="text-11px opacity-50">{{ formatFileSize(file.fileSize) }}</span>
      <span v-else class="text-11px opacity-50">{{ $t('page.disk.file.folder') }}</span>
    </template>
  </div>
</template>

<style scoped>
.disk-card-active {
  background-color: rgba(var(--primary-color-rgb), 0.1);
}
</style>
