<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { fetchUpgradeStatus } from '@/service/api/system';

defineOptions({ name: 'UpgradeProgressDialog' });

interface Props {
  visible: boolean;
}

interface Emits {
  (e: 'update:visible', v: boolean): void;
  /** 升级流程结束（成功或失败）时通知，调用方可据此刷新版本信息 */
  (e: 'finished', success: boolean): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const { t } = useI18n();

const status = ref<Api.System.UpgradeStateInfo | null>(null);
let pollTimer: ReturnType<typeof setInterval> | null = null;

/** 状态机阶段 → 文案/图标/是否进行中（computed 内取 t，切语言联动） */
const stage = computed(() => {
  const s = status.value?.state ?? 'idle';
  const map: Record<string, { text: string; icon: string; processing: boolean; tone: 'default' | 'success' | 'error' | 'warning' }> = {
    idle: { text: t('upgrade.stageIdle'), icon: 'ph:info', processing: false, tone: 'default' },
    downloading: { text: t('upgrade.stageDownloading'), icon: 'ph:download-simple', processing: true, tone: 'default' },
    verifying: { text: t('upgrade.stageVerifying'), icon: 'ph:shield-check', processing: true, tone: 'default' },
    unpacking: { text: t('upgrade.stageUnpacking'), icon: 'ph:package', processing: true, tone: 'default' },
    installing: { text: t('upgrade.stageInstalling'), icon: 'ph:rocket-launch', processing: true, tone: 'default' },
    success: { text: t('upgrade.stageSuccess'), icon: 'ph:check-circle', processing: false, tone: 'success' },
    failed: { text: t('upgrade.stageFailed'), icon: 'ph:x-circle', processing: false, tone: 'error' },
    unreachable: { text: t('upgrade.stageUnreachable'), icon: 'ph:wifi-slash', processing: false, tone: 'warning' }
  };
  return map[s] ?? map.idle;
});

const isTerminal = computed(() => ['success', 'failed', 'unreachable'].includes(status.value?.state ?? ''));

/** 仅 downloading 有真实下载百分比（终态 success 满格，其余进行中阶段不显示进度条只转圈） */
const progressPercent = computed(() => {
  const s = status.value?.state ?? '';
  if (s === 'downloading') return Math.min(100, Math.max(0, status.value?.progress ?? 0));
  if (s === 'success') return 100;
  return null;
});

function stopPoll() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

async function pollStatus() {
  const { data, error } = await fetchUpgradeStatus();
  if (!error && data) {
    const wasProcessing = !isTerminal.value;
    status.value = data;
    if (wasProcessing && isTerminal.value) {
      emit('finished', data.state === 'success');
    }
  }
}

watch(
  () => props.visible,
  v => {
    stopPoll();
    if (v) {
      status.value = null;
      pollStatus();
      pollTimer = setInterval(pollStatus, 3000);
    }
  }
);

onBeforeUnmount(stopPoll);

function handleClose() {
  stopPoll();
  emit('update:visible', false);
}
</script>

<template>
  <NModal
    :show="visible"
    preset="card"
    :title="t('upgrade.progressTitle')"
    class="w-480px"
    :mask-closable="false"
    :closable="isTerminal"
    @update:show="(v: boolean) => (v || isTerminal) && handleClose()"
  >
    <div class="flex flex-col gap-16px px-8px py-8px">
      <!-- 阶段 + 目标版本 -->
      <div class="flex items-center gap-10px">
        <NSpin v-if="stage.processing" :size="20" />
        <SvgIcon
          v-else
          :icon="stage.icon"
          class="text-20px"
          :class="{ 'text-success': stage.tone === 'success', 'text-error': stage.tone === 'error', 'text-warning': stage.tone === 'warning' }"
        />
        <span class="text-15px font-600">{{ stage.text }}</span>
        <span v-if="status?.version" class="text-13px opacity-60">v{{ status.version }}</span>
      </div>

      <!-- 下载百分比进度条（其余进行中阶段以转圈表达，避免假进度） -->
      <NProgress
        v-if="progressPercent !== null"
        type="line"
        :percentage="progressPercent"
        :status="status?.state === 'failed' ? 'error' : undefined"
      />

      <!-- 进展/错误说明 -->
      <div v-if="status?.message" class="text-13px opacity-70 break-all">{{ status.message }}</div>

      <!-- 终态提示 -->
      <NAlert v-if="status?.state === 'success'" type="success" :show-icon="false" class="text-13px">
        {{ t('upgrade.successTip') }}
      </NAlert>
      <NAlert v-else-if="status?.state === 'unreachable'" type="warning" :show-icon="false" class="text-13px">
        {{ t('upgrade.unreachableTip') }}
      </NAlert>

      <div v-if="!isTerminal" class="text-12px opacity-50">{{ t('upgrade.processingTip') }}</div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-12px">
        <NButton v-if="!isTerminal" quaternary @click="handleClose">{{ t('upgrade.background') }}</NButton>
        <NButton v-else type="primary" @click="handleClose">{{ t('common.close') }}</NButton>
      </div>
    </template>
  </NModal>
</template>
