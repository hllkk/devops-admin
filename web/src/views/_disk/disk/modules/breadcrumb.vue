<script setup lang="ts">
import { computed } from 'vue';
import { useDiskStore } from '@/store/modules/disk';
import { $t } from '@/locales';

defineOptions({ name: 'DiskBreadcrumb' });

interface Props {
  totalCount?: number;
}
const props = defineProps<Props>();

const diskStore = useDiskStore();

interface Crumb {
  /** -1 表示根目录 */
  index: number;
  label: string;
}

const crumbs = computed<Crumb[]>(() => [
  { index: -1, label: $t('page.disk.breadcrumb.root') },
  ...diskStore.currentPath.map((f, i) => ({ index: i, label: f.fileName }))
]);

function handleClick(c: Crumb) {
  if (c.index === -1) {
    diskStore.resetPath();
  } else {
    diskStore.goBack(c.index);
  }
}
</script>

<template>
  <div class="flex-y-center gap-4px px-12px py-6px text-13px">
    <!-- 返回上一级:进入子目录后才显示,位于面包屑最前 -->
    <template v-if="diskStore.currentPath.length > 0">
      <span class="flex-y-center gap-2px cursor-pointer hover:text-primary" @click="diskStore.goBack()">
        <SvgIcon icon="material-symbols:arrow-upward" class="text-16px" />
        {{ $t('page.disk.breadcrumb.backToPrev') }}
      </span>
      <span class="opacity-30">|</span>
    </template>
    <!-- 路径:全部文件 > 文件夹A > 文件夹B -->
    <template v-for="(c, i) in crumbs" :key="`${c.index}-${i}`">
      <SvgIcon v-if="i > 0" icon="material-symbols:chevron-right" class="text-16px opacity-50" />
      <span
        class="cursor-pointer hover:text-primary"
        :class="{ 'font-500': i === crumbs.length - 1 }"
        @click="handleClick(c)"
      >
        {{ c.label }}
      </span>
    </template>
    <span v-if="props.totalCount !== undefined" class="ml-auto opacity-50">
      {{ $t('page.disk.breadcrumb.count', { count: props.totalCount }) }}
    </span>
  </div>
</template>
