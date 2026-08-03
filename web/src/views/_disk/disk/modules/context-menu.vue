<script setup lang="ts">
import { computed } from 'vue';
import type { DropdownOption } from 'naive-ui';
import { useSvgIcon } from '@/hooks/common/icon';
import { $t } from '@/locales';

defineOptions({
  name: 'DiskContextMenu'
});

interface Props {
  x: number;
  y: number;
  type: 'file' | 'area';
  /** 当前文件是否已收藏(右键单个文件时,控制收藏项文案与图标) */
  fileIsFavorite?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  fileIsFavorite: false
});

const visible = defineModel<boolean>('visible');

const emit = defineEmits<{
  (e: 'select', key: string): void;
}>();

const { SvgIconVNode } = useSvgIcon();

/**
 * 本地暂缺下游的菜单项先禁用,待对应功能(打开/预览、收藏、分享、详情)补齐后,
 * 从该集合移除对应 key 即可启用,无需改动菜单结构。
 */
const PENDING_KEYS = new Set(['open', 'addFavorite', 'removeFavorite', 'share', 'detail']);

const fileOptions = computed<DropdownOption[]>(() => {
  const showRemoveFavorite = props.fileIsFavorite === true;
  const favoriteKey = showRemoveFavorite ? 'removeFavorite' : 'addFavorite';

  return [
    {
      label: $t('page.disk.contextMenu.open'),
      key: 'open',
      disabled: PENDING_KEYS.has('open'),
      icon: SvgIconVNode({ icon: 'material-symbols:open-in-new', fontSize: 18 })
    },
    {
      label: showRemoveFavorite ? $t('page.disk.contextMenu.removeFavorite') : $t('page.disk.contextMenu.addFavorite'),
      key: favoriteKey,
      disabled: PENDING_KEYS.has(favoriteKey),
      icon: SvgIconVNode({
        icon: showRemoveFavorite ? 'material-symbols:star' : 'material-symbols:star-outline',
        fontSize: 18
      })
    },
    {
      label: $t('page.disk.contextMenu.download'),
      key: 'download',
      icon: SvgIconVNode({ icon: 'material-symbols:download', fontSize: 18 })
    },
    {
      label: $t('page.disk.contextMenu.share'),
      key: 'share',
      disabled: PENDING_KEYS.has('share'),
      icon: SvgIconVNode({ icon: 'material-symbols:ios-share', fontSize: 18 })
    },
    {
      label: $t('page.disk.contextMenu.copy'),
      key: 'copy',
      icon: SvgIconVNode({ icon: 'material-symbols:content-copy', fontSize: 18 })
    },
    {
      label: $t('page.disk.contextMenu.move'),
      key: 'move',
      icon: SvgIconVNode({ icon: 'material-symbols:drive-file-move-outline', fontSize: 18 })
    },
    {
      label: $t('page.disk.contextMenu.rename'),
      key: 'rename',
      icon: SvgIconVNode({ icon: 'material-symbols:edit', fontSize: 18 })
    },
    {
      label: $t('page.disk.contextMenu.delete'),
      key: 'delete',
      icon: SvgIconVNode({ icon: 'material-symbols:delete-outline', fontSize: 18 })
    },
    { type: 'divider', key: 'd-detail' },
    {
      label: $t('page.disk.contextMenu.detail'),
      key: 'detail',
      disabled: PENDING_KEYS.has('detail'),
      icon: SvgIconVNode({ icon: 'material-symbols:info', fontSize: 18 })
    }
  ];
});

const areaOptions = computed<DropdownOption[]>(() => [
  {
    label: $t('page.disk.contextMenu.view'),
    key: 'view',
    icon: SvgIconVNode({ icon: 'material-symbols:visibility-outline', fontSize: 18 }),
    children: [
      {
        label: $t('page.disk.contextMenu.viewGrid'),
        key: 'view-grid',
        icon: SvgIconVNode({ icon: 'material-symbols:grid-view', fontSize: 18 })
      },
      {
        label: $t('page.disk.contextMenu.viewList'),
        key: 'view-list',
        icon: SvgIconVNode({ icon: 'material-symbols:view-list', fontSize: 18 })
      }
    ]
  },
  {
    label: $t('page.disk.contextMenu.sortBy'),
    key: 'sort',
    icon: SvgIconVNode({ icon: 'material-symbols:sort', fontSize: 18 }),
    children: [
      {
        label: $t('page.disk.contextMenu.sortName'),
        key: 'sort-name',
        icon: SvgIconVNode({ icon: 'material-symbols:abc', fontSize: 18 })
      },
      {
        label: $t('page.disk.contextMenu.sortSize'),
        key: 'sort-size',
        icon: SvgIconVNode({ icon: 'material-symbols:database', fontSize: 18 })
      },
      {
        label: $t('page.disk.contextMenu.sortTime'),
        key: 'sort-modifyTime',
        icon: SvgIconVNode({ icon: 'material-symbols:schedule', fontSize: 18 })
      }
    ]
  },
  {
    label: $t('page.disk.contextMenu.refresh'),
    key: 'refresh',
    icon: SvgIconVNode({ icon: 'material-symbols:refresh', fontSize: 18 })
  },
  {
    label: $t('page.disk.contextMenu.reload'),
    key: 'reload',
    icon: SvgIconVNode({ icon: 'material-symbols:sync', fontSize: 18 })
  }
]);

const options = computed(() => (props.type === 'file' ? fileOptions.value : areaOptions.value));

function hide() {
  visible.value = false;
}

function handleSelect(key: string) {
  emit('select', key);
  hide();
}
</script>

<template>
  <NDropdown
    :show="visible"
    placement="bottom-start"
    trigger="manual"
    :x="x"
    :y="y"
    :options="options"
    :menu-props="() => ({ class: 'disk-ctx-glass' })"
    @clickoutside="hide"
    @select="handleSelect"
  />
</template>

<style>
/* 右键菜单毛玻璃效果(全局:NDropdown teleport 到 body,需非 scoped) */
.n-dropdown-menu.disk-ctx-glass {
  background: rgba(255, 255, 255, 0.55) !important;
  backdrop-filter: blur(20px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 10px;
}

html.dark .n-dropdown-menu.disk-ctx-glass {
  background: rgba(30, 30, 30, 0.6) !important;
  border-color: rgba(255, 255, 255, 0.08);
}
</style>
