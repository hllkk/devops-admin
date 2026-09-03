<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import SvgIcon from '@/components/custom/svg-icon.vue';

defineOptions({ name: 'IconUrl' });

/**
 * 按 iconUrl 字段值渲染图标：
 * - 本地图标名(如 ai-bot)走 SvgIcon symbol 体系
 * - http(s)/绝对路径走 <img>，加载失败回退本地默认图标(兼容历史手输 URL 数据)
 * - 空值渲染项目默认 no-icon
 */
interface Props {
  value?: string | null;
  /** 渲染尺寸(px)，本地图标经 font-size 生效 */
  size?: number;
}

const props = withDefaults(defineProps<Props>(), {
  value: '',
  size: 20
});

const isRemote = computed(() => /^(https?:)?\/\//.test(props.value || '') || (props.value || '').startsWith('/'));

const localIcon = computed(() => (isRemote.value ? 'no-icon' : props.value || 'no-icon'));

/** img 加载失败置位后转本地默认图标渲染 */
const imgFailed = ref(false);

watch(
  () => props.value,
  () => {
    imgFailed.value = false;
  }
);
</script>

<template>
  <SvgIcon
    v-if="!isRemote || imgFailed"
    :local-icon="localIcon"
    class="shrink-0"
    :style="{ fontSize: `${size}px` }"
  />
  <img
    v-else
    :src="value!"
    :width="size"
    :height="size"
    alt=""
    class="shrink-0 rounded object-contain"
    @error="imgFailed = true"
  />
</template>

<style scoped></style>
