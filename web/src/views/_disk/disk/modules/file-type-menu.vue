<script setup lang="ts">
import { computed, ref } from 'vue';
import { $t } from '@/locales';
import { useDiskStore } from '@/store/modules/disk';
import SvgIcon from '@/components/custom/svg-icon.vue';
import { formatFileSize } from '@/utils/format';
import { useSvgIcon } from '@/hooks/common/icon';

defineOptions({ name: 'FileTypeMenu' });

const { SvgIconVNode } = useSvgIcon();

interface Props {
  /** 显示容量区块 */
  showCapacity?: boolean;
  /** 配额信息 */
  quotaInfo?: Api.Disk.QuotaInfo;
  /** 配额加载中 */
  quotaLoading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  showCapacity: false,
  quotaInfo: () => ({ usedSpace: 0, quota: 0, unlimited: false, quotaSource: 'none' }),
  quotaLoading: false
});

const diskStore = useDiskStore();

/** 文件类型列表（localIcon 对应 src/assets/svg-icon/disk/*.svg，symbolId = icon-local-disk-<name>） */
const fileTypeList: Array<{ name: string; value: Api.Disk.FileType; localIcon: string }> = [
  { name: $t('page.disk.fileType.all'), value: 'all', localIcon: 'disk-menu-file' },
  { name: $t('page.disk.fileType.image'), value: 'image', localIcon: 'disk-file-image' },
  { name: $t('page.disk.fileType.document'), value: 'document', localIcon: 'disk-file-txt' },
  { name: $t('page.disk.fileType.video'), value: 'video', localIcon: 'disk-file-video' },
  { name: $t('page.disk.fileType.audio'), value: 'audio', localIcon: 'disk-file-music' },
  { name: $t('page.disk.fileType.other'), value: 'other', localIcon: 'disk-file-other' }
];

/** 当前选中（双向，读 diskStore） */
const selectedType = computed({
  get: () => diskStore.currentFileType,
  set: (val: Api.Disk.FileType) => diskStore.setFileType(val)
});

/** NMenu options：单分组「文件类型」，children 为各类型 */
const menuOptions = computed(() => [
  {
    key: 'file-types',
    label: $t('page.disk.fileType.title'),
    icon: SvgIconVNode({ localIcon: 'disk-list-folder', fontSize: 24 }),
    children: fileTypeList.map(item => ({
      key: item.value,
      label: item.name,
      icon: SvgIconVNode({ localIcon: item.localIcon, fontSize: 30 })
    }))
  }
]);

/** 默认展开分组 */
const expandedKeys = ref<string[]>(['file-types']);

function handleMenuSelect(key: string) {
  if (key !== 'file-types') {
    selectedType.value = key as Api.Disk.FileType;
  }
}

// === 容量 ===
const usedPercent = computed(() => {
  if (props.quotaInfo.unlimited || props.quotaInfo.quota === 0) return 0;
  return Math.min(100, Math.round((props.quotaInfo.usedSpace / props.quotaInfo.quota) * 100));
});

const capacityData = computed(() => ({
  used: formatFileSize(props.quotaInfo.usedSpace),
  total: props.quotaInfo.unlimited ? $t('page.disk.capacity.unlimited') : formatFileSize(props.quotaInfo.quota)
}));

/** 进度颜色随用量分级 */
const capacityColor = computed(() => {
  const p = usedPercent.value;
  if (p < 60) return '#18a058';
  if (p < 85) return '#f0a020';
  return '#d03050';
});

/** 容量环 conic-gradient —— 动态角度经 :style 绑定，后半透明露出底层 bg 作未用色 */
const ringGradient = computed(() => {
  const deg = usedPercent.value * 3.6;
  return `conic-gradient(${capacityColor.value} 0deg, ${capacityColor.value} ${deg}deg, transparent ${deg}deg, transparent 360deg)`;
});
</script>

<template>
  <div class="flex flex-col h-full">
    <NMenu
      v-model:expanded-keys="expandedKeys"
      :value="selectedType"
      :options="menuOptions"
      :indent="14"
      class="flex-1 select-none"
      @update:value="handleMenuSelect"
    />

    <!-- 容量区块（受开关控制，钉底部） -->
    <div
      v-if="props.showCapacity"
      class="mt-auto flex items-center gap-16px p-12px rd-12px bg-gradient-to-br from-green-500/8 to-blue-500/8 dark:from-green-500/15 dark:to-blue-500/15 dark:border dark:border-white/10"
    >
      <NSkeleton v-if="props.quotaLoading" text :repeat="3" />
      <template v-else>
        <!-- 无限制 -->
        <div v-if="props.quotaInfo.unlimited" class="flex items-center gap-6px text-14px font-500">
          <SvgIcon icon="mdi:cloud-outline" class="text-20px text-primary" />
          <span>{{ $t('page.disk.capacity.title') }}：{{ $t('page.disk.capacity.unlimited') }}</span>
        </div>
        <template v-else>
          <!-- 容量环：三层 absolute 叠加，动态角度经 :style 绑定 -->
          <div class="relative size-80px">
            <div class="absolute inset-0 rd-full bg-black/8 dark:bg-white/12 shadow-[inset_0_0_10px_rgba(0,0,0,0.15)] dark:shadow-[inset_0_0_10px_rgba(255,255,255,0.15)]" />
            <div class="absolute inset-4px rd-full" :style="{ background: ringGradient }" />
            <div class="absolute inset-12px flex-center rd-full bg-[var(--n-color)]">
              <span class="text-22px font-600">{{ usedPercent }}<span class="ml-2px text-12px opacity-60">%</span></span>
            </div>
          </div>
          <!-- 用量 -->
          <div class="flex flex-1 flex-col gap-10px">
            <div class="flex items-center gap-6px text-14px font-500">
              <SvgIcon icon="mdi:cloud-outline" class="text-20px text-primary" />
              <span>{{ $t('page.disk.capacity.title') }}</span>
            </div>
            <div class="flex items-center justify-between text-13px">
              <span class="opacity-60">{{ $t('page.disk.capacity.usedLabel') }}</span>
              <span class="font-600" :style="{ color: capacityColor }">{{ capacityData.used }}</span>
            </div>
            <div class="flex items-center justify-between text-13px">
              <span class="opacity-60">{{ $t('page.disk.capacity.totalLabel') }}</span>
              <span class="font-600">{{ capacityData.total }}</span>
            </div>
          </div>
        </template>
      </template>
    </div>
  </div>
</template>

<style scoped>
/* NMenu 图标列宽：子项图标经 SvgIconVNode({ fontSize: 30 }) 渲染，Naive 默认图标列宽偏窄会裁切，
   :deep 放宽 .n-menu-item-content__icon 宽度到 48px 给足空间 */
:deep(.n-menu-item-content) {
  .n-menu-item-content__icon {
    width: 48px !important;
  }
}
</style>
