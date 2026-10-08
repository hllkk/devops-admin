<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue';
import { fetchGetInstallStatus, fetchInstallAgent } from '@/service/api/server';
import { $t } from '@/locales';

defineOptions({ name: 'AgentInstallModal' });

interface Props {
  /** 目标资产(physical/vm,已通过 SSH 录入验证) */
  row: Api.Server.Asset | null;
}

const props = defineProps<Props>();

interface Emits {
  (e: 'finished'): void;
}

const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', { default: false });

const submitting = ref(false);

// 安装进度(提交后轮询)
const polling = ref(false);
const installStatus = ref<Api.Server.AgentInstallStatus | null>(null);
let pollTimer: ReturnType<typeof setInterval> | null = null;

const modalTitle = computed(() =>
  props.row ? $t('page.server.agentInstall.title', { name: props.row.assetName }) : $t('page.server.agentInstall.titlePlain')
);

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
  polling.value = false;
}

function reset() {
  stopPolling();
  installStatus.value = null;
}

async function startPolling(taskId: string) {
  polling.value = true;
  pollTimer = setInterval(async () => {
    const { error, data } = await fetchGetInstallStatus(taskId);
    if (!error && data) {
      installStatus.value = data;
      if (data.status !== 'running') {
        stopPolling();
        if (data.status === 'success') {
          window.$message?.success($t('page.server.agentInstall.success'));
          emit('finished');
        } else {
          window.$message?.error($t('page.server.agentInstall.failed') + (data.message ? `: ${data.message}` : ''));
        }
      }
    }
  }, 2000);
}

async function handleSubmit() {
  if (!props.row) return;
  submitting.value = true;
  const { error, data } = await fetchInstallAgent(props.row.assetId!);
  submitting.value = false;
  if (error) return;
  window.$message?.success($t('page.server.agentInstall.started'));
  if (data?.taskId) {
    startPolling(data.taskId);
  }
}

onUnmounted(() => {
  stopPolling();
});
</script>

<template>
  <NModal
    v-model:show="visible"
    preset="card"
    :title="modalTitle"
    class="w-480px"
    :mask-closable="!polling"
    @after-leave="reset"
  >
    <NAlert v-if="!installStatus" type="info" :show-icon="true">
      {{ $t('page.server.agentInstall.form.notice') }}
    </NAlert>

    <!-- 安装进度 -->
    <div v-if="installStatus" class="flex flex-col gap-8px">
      <NDivider>{{ $t('page.server.agentInstall.progress') }}</NDivider>
      <div class="flex items-center gap-8px">
        <NSpin v-if="installStatus.status === 'running'" :size="16" />
        <NTag v-else :type="installStatus.status === 'success' ? 'success' : 'error'" size="small">
          {{ $t(`page.server.agentInstall.status_${installStatus.status}`) }}
        </NTag>
        <span class="text-13px">{{ installStatus.step }}</span>
      </div>
      <span v-if="installStatus.message" class="text-12px" :class="installStatus.status === 'failed' ? 'color-error' : 'text-slate-400'">
        {{ installStatus.message }}
      </span>
    </div>

    <template #footer>
      <NSpace class="w-full" justify="end">
        <NButton @click="visible = false">{{ $t(polling ? 'common.close' : 'common.cancel') }}</NButton>
        <NButton v-if="!installStatus || installStatus.status === 'failed'" type="primary" :loading="submitting" @click="handleSubmit">
          {{ $t('page.server.agentInstall.submit') }}
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped></style>
