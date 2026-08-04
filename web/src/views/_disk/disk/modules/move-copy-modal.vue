<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import type { TreeOption } from 'naive-ui';
import { $t } from '@/locales';
import { fetchFolderTree } from '@/service/api/disk';

defineOptions({ name: 'MoveCopyModal' });

interface Props {
  visible: boolean;
  mode: 'move' | 'copy';
  sources: Api.Disk.FileItem[];
}

interface Emits {
  (e: 'confirm', targetPath: string): void;
  (e: 'update:visible', v: boolean): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const treeData = ref<TreeOption[]>([]);
const selectedKeys = ref<string[]>(['/']);
const loading = ref(false);

/** 源项标签:单个显示文件名,多个显示"已选中 N 个" */
const sourceLabel = computed(() => {
  const list = props.sources;
  if (list.length === 0) return '';
  if (list.length === 1) return list[0].fileName;
  return $t('page.disk.column.selectedCount', { count: list.length });
});

watch(
  () => props.visible,
  async v => {
    if (!v) return;
    selectedKeys.value = ['/'];
    await loadTree();
  }
);

async function loadTree() {
  loading.value = true;
  const { data, error } = await fetchFolderTree();
  loading.value = false;
  if (!error && data) {
    // 前置一个"根目录"节点(可移/复制到根);树节点 key 直接用 path,选中即目标路径
    treeData.value = [{ key: '/', label: $t('page.disk.breadcrumb.root'), children: data.map(toTreeOption) }];
  } else {
    treeData.value = [];
  }
}

function toTreeOption(n: Api.Disk.FolderTreeNode): TreeOption {
  return { key: n.path, label: n.name, children: n.children?.map(toTreeOption) };
}

function handleSelect(keys: Array<string | number>) {
  selectedKeys.value = keys.map(k => String(k));
}

function close() {
  emit('update:visible', false);
}

function confirm() {
  const target = selectedKeys.value[0];
  if (target !== undefined) emit('confirm', target);
}
</script>

<template>
  <NModal
    :show="visible"
    preset="card"
    :title="mode === 'move' ? $t('page.disk.modal.moveTitle') : $t('page.disk.modal.copyTitle')"
    class="w-420px"
    :mask-closable="false"
    @update:show="(v: boolean) => emit('update:visible', v)"
  >
    <div class="mb-4px text-13px">
      <span class="opacity-70">{{ sourceLabel }}</span>
      <span class="mx-4px opacity-40">→</span>
      <span class="opacity-70">{{ $t('page.disk.modal.targetLabel') }}</span>
    </div>
    <NSpin :show="loading">
      <div class="max-h-320px overflow-auto">
        <NTree
          :data="treeData"
          :selected-keys="selectedKeys"
          :default-expanded-keys="['/']"
          selectable
          block-line
          @update:selected-keys="handleSelect"
        />
      </div>
    </NSpin>
    <template #footer>
      <div class="flex justify-end gap-12px">
        <NButton @click="close">{{ $t('page.disk.modal.cancel') }}</NButton>
        <NButton type="primary" @click="confirm">{{ $t('page.disk.modal.confirm') }}</NButton>
      </div>
    </template>
  </NModal>
</template>
