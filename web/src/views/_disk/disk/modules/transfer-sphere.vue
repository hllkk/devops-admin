<script setup lang="ts">
import { computed } from 'vue';

defineOptions({ name: 'TransferSphere' });

const props = defineProps<{
  progress: number; // 0-100
  done: boolean;
}>();

const R = 56;
const circ = computed(() => 2 * Math.PI * R);
const pct = computed(() => Math.min(100, Math.max(0, props.progress)));
// 水位:圆 r=48,中心 (60,60),顶=12 底=108。progress 0→水面在底(108),100→在顶(12)
const waterY = computed(() => 108 - (pct.value / 100) * 96);
const dash = computed(() => `${(pct.value / 100) * circ.value} ${circ.value}`);
</script>

<template>
  <div class="disk-sphere relative h-160px w-160px">
    <svg viewBox="0 0 120 120" class="h-full w-full">
      <defs>
        <clipPath id="diskSphereClip">
          <circle cx="60" cy="60" r="48" />
        </clipPath>
      </defs>
      <!-- 进度环底 -->
      <circle cx="60" cy="60" :r="R" fill="none" stroke="rgb(var(--primary-color) / 0.12)" stroke-width="4" />
      <!-- 进度环 -->
      <circle
        cx="60"
        cy="60"
        :r="R"
        fill="none"
        stroke="rgb(var(--primary-color))"
        stroke-width="4"
        stroke-linecap="round"
        :stroke-dasharray="dash"
        transform="rotate(-90 60 60)"
      />
      <!-- 液体(裁剪在圆内) -->
      <g clip-path="url(#diskSphereClip)">
        <g :transform="`translate(0 ${waterY})`">
          <g>
            <path
              d="M0,6 q15,-12 30,0 t30,0 t30,0 t30,0 t30,0 t30,0 t30,0 t30,0 L240,140 L0,140 Z"
              fill="rgb(var(--primary-color))"
              opacity="0.55"
            />
            <animateTransform attributeName="transform" type="translate" from="0 0" to="-60 0" dur="3s" repeatCount="indefinite" />
          </g>
          <g>
            <path
              d="M0,10 q15,-12 30,0 t30,0 t30,0 t30,0 t30,0 t30,0 t30,0 t30,0 L240,140 L0,140 Z"
              fill="rgb(var(--primary-color))"
              opacity="0.3"
            />
            <animateTransform attributeName="transform" type="translate" from="0 0" to="-60 0" dur="4.5s" repeatCount="indefinite" />
          </g>
        </g>
      </g>
      <!-- 内圆描边 -->
      <circle cx="60" cy="60" r="48" fill="none" stroke="rgb(var(--primary-color) / 0.2)" stroke-width="1" />
    </svg>
    <div class="pointer-events-none absolute inset-0 flex-col-center">
      <SvgIcon v-if="done" icon="material-symbols:check-circle" class="text-30px" style="color: rgb(var(--primary-color))" />
      <span v-else class="text-22px font-600 tabular-nums">{{ pct }}%</span>
    </div>
  </div>
</template>

<style scoped>
.disk-sphere {
  filter: drop-shadow(0 0 8px rgb(var(--primary-color) / 0.25));
}
</style>
