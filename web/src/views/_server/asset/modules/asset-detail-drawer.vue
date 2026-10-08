<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue';
import { NDescriptions, NDescriptionsItem, NTag, NTime } from 'naive-ui';
import {
  fetchGetAssetSnapshot,
  fetchGetInstallStatus,
  fetchRestartAgent,
  fetchUninstallAgent
} from '@/service/api/server';
import { $t } from '@/locales';
import { ASSET_TYPE_OPTIONS, MONITOR_STATUS_OPTIONS, AGENT_STATUS_OPTIONS } from '@/constants/business/server';
import AgentInstallModal from './agent-install-modal.vue';

defineOptions({ name: 'AssetDetailDrawer' });

interface Props {
  row: Api.Server.Asset | null;
}

const props = defineProps<Props>();

const visible = defineModel<boolean>('visible', { default: false });

// 数据刷新:打开/操作完成时通知父组件刷新列表(资产行状态变了)
const emit = defineEmits<{
  (e: 'changed'): void;
}>();

// ── 快照(agent 心跳指标) ──
const snapshot = ref<Api.Server.AgentSnapshot | null>(null);
const snapshotLoading = ref(false);
let snapTimer: ReturnType<typeof setInterval> | null = null;

async function loadSnapshot() {
  if (!props.row) return;
  snapshotLoading.value = true;
  const { error, data } = await fetchGetAssetSnapshot(props.row.assetId!);
  snapshotLoading.value = false;
  if (!error && data) {
    snapshot.value = data;
  }
}

function stopSnapTimer() {
  if (snapTimer) {
    clearInterval(snapTimer);
    snapTimer = null;
  }
}

watch(visible, v => {
  stopSnapTimer();
  if (v) {
    loadSnapshot();
    // 打开期间 30s 刷新一次快照(与 agent 心跳同步频)
    snapTimer = setInterval(loadSnapshot, 30000);
  }
});
onUnmounted(stopSnapTimer);

// ── 标签/文案 ──
const assetTypeLabel = computed(() => {
  const o = ASSET_TYPE_OPTIONS.find(x => x.value === props.row?.assetType);
  return o ? $t(o.label) : props.row?.assetType ?? '-';
});
const agentStatusLabel = computed(() => {
  const v = props.row?.agentStatus;
  const o = AGENT_STATUS_OPTIONS.find(x => x.value === v);
  return o ? $t(o.label) : v ?? '-';
});
const monitorStatusLabel = computed(() => {
  const v = props.row?.monitorStatus;
  const o = MONITOR_STATUS_OPTIONS.find(x => x.value === v);
  return o ? $t(o.label) : v ?? '-';
});
const monitorTagType = computed(() => {
  const v = props.row?.monitorStatus;
  if (v === 'online') return 'success';
  if (v === 'offline') return 'error';
  return 'default';
});
const agentTagType = computed(() => {
  const v = props.row?.agentStatus;
  if (v === 'running') return 'success';
  if (v === 'lost') return 'error';
  if (v === 'installing') return 'warning';
  return 'default';
});

function fmtKB(kb: number): string {
  if (!kb) return '-';
  if (kb < 1024 * 1024) return `${(kb / 1024).toFixed(1)} MB`;
  return `${(kb / 1024 / 1024).toFixed(1)} GB`;
}
function fmtBytes(b: number): string {
  if (!b) return '-';
  if (b < 1024) return `${b} B`;
  if (b < 1024 ** 3) return `${(b / 1024 ** 2).toFixed(1)} MB`;
  return `${(b / 1024 ** 3).toFixed(1)} GB`;
}
function fmtBps(bps: number): string {
  if (!bps) return '-';
  if (bps < 1024) return `${bps.toFixed(0)} B/s`;
  if (bps < 1024 ** 2) return `${(bps / 1024).toFixed(1)} KB/s`;
  return `${(bps / 1024 ** 2).toFixed(1)} MB/s`;
}
function fmtUptime(sec: number): string {
  if (!sec) return '-';
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  if (d > 0) return `${d}天${h}小时`;
  if (h > 0) return `${h}小时${m}分`;
  return `${m}分`;
}
function memUsedPercent(s: Api.Server.AgentSnapshot | null): number {
  if (!s || !s.memTotalKb) return 0;
  return ((s.memTotalKb - s.memAvailableKb) / s.memTotalKb) * 100;
}
function diskUsedPercent(s: Api.Server.AgentSnapshot | null): number {
  if (!s || !s.diskTotalBytes) return 0;
  return (s.diskUsedBytes / s.diskTotalBytes) * 100;
}

const isSSHAsset = computed(() => ['physical', 'vm'].includes(props.row?.assetType ?? ''));

/** 最近心跳时间戳(ms,0=无) */
const lastHeartbeatTime = computed(() => {
  const at = props.row?.lastHeartbeatAt;
  return at ? Date.parse(at) || 0 : 0;
});
const agentInstalled = computed(() => {
  const v = props.row?.agentStatus;
  return !!v && v !== 'none';
});

// ── 运维操作(重启/卸载:异步任务+轮询;安装走弹窗) ──
const opsRunning = ref(false);
const opsStatus = ref<Api.Server.AgentInstallStatus | null>(null);
let opsTimer: ReturnType<typeof setInterval> | null = null;

const installModalVisible = ref(false);

function stopOpsTimer() {
  if (opsTimer) {
    clearInterval(opsTimer);
    opsTimer = null;
  }
}

function startOpsPolling(taskId: string) {
  opsRunning.value = true;
  opsTimer = setInterval(async () => {
    const { error, data } = await fetchGetInstallStatus(taskId);
    if (!error && data) {
      opsStatus.value = data;
      if (data.status !== 'running') {
        stopOpsTimer();
        opsRunning.value = false;
        if (data.status === 'success') {
          window.$message?.success(data.message || $t('common.updateSuccess'));
          emit('changed');
          loadSnapshot();
        } else {
          window.$message?.error(`${$t('page.server.agentOps.opsFailed')}${data.message ? `: ${data.message}` : ''}`);
        }
      }
    }
  }, 2000);
}

async function triggerOps(kind: 'restart' | 'uninstall') {
  if (!props.row) return;
  window.$dialog?.warning({
    title: $t(kind === 'restart' ? 'page.server.agentOps.restartConfirm' : 'page.server.agentOps.uninstallConfirm'),
    content: $t(kind === 'restart' ? 'page.server.agentOps.restartConfirmDesc' : 'page.server.agentOps.uninstallConfirmDesc'),
    positiveText: $t('common.confirm'),
    negativeText: $t('common.cancel'),
    onPositiveClick: async () => {
      opsStatus.value = null;
      const { error, data } =
        kind === 'restart' ? await fetchRestartAgent(props.row!.assetId!) : await fetchUninstallAgent(props.row!.assetId!);
      if (error) return;
      if (data?.taskId) startOpsPolling(data.taskId);
    }
  });
}

function handleInstallFinished() {
  emit('changed');
}

watch(visible, v => {
  if (!v) {
    stopOpsTimer();
    opsRunning.value = false;
    opsStatus.value = null;
  }
});
</script>

<template>
  <NDrawer v-model:show="visible" :width="520" placement="right">
    <NDrawerContent :title="$t('page.server.assetDetail.title')" closable>
      <!-- 基本信息 -->
      <NDescriptions v-if="row" :column="2" label-placement="left" bordered size="small" class="mb-16px">
        <NDescriptionsItem :label="$t('page.server.asset.col.assetName')" :span="2">
          <span class="font-500">{{ row.assetName }}</span>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.asset.col.assetType')">
          <NTag size="small">{{ assetTypeLabel }}</NTag>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.asset.col.manageIp')">{{ row.manageIp || '-' }}</NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.asset.col.sshPort')">
          {{ isSSHAsset ? row.sshPort : '-' }}
        </NDescriptionsItem>
        <NDescriptionsItem v-if="isSSHAsset" :label="$t('page.server.asset.col.sshUsername')">
          {{ row.sshUsername || '-' }}
        </NDescriptionsItem>
        <NDescriptionsItem v-if="isSSHAsset" :label="$t('page.server.asset.col.sshVerified')">
          <NTag size="small" :type="row.sshVerified ? 'success' : 'warning'" :bordered="false">
            {{ row.sshVerified ? $t('page.server.asset.sshVerified') : $t('page.server.asset.sshUnverified') }}
          </NTag>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.asset.col.osType')">{{ row.osType || '-' }}</NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.asset.col.env')">{{ row.env || '-' }}</NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.asset.col.location')" :span="2">
          {{ row.location || '-' }}
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.asset.col.isActive')">
          <NTag size="small" :type="row.isActive ? 'success' : 'default'" :bordered="false">
            {{ row.isActive ? $t('page.server.common.active') : $t('page.server.common.inactive') }}
          </NTag>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.asset.col.monitorStatus')">
          <NTag size="small" :type="monitorTagType" :bordered="false">{{ monitorStatusLabel }}</NTag>
        </NDescriptionsItem>
        <NDescriptionsItem v-if="row.description" :label="$t('page.server.asset.col.description')" :span="2">
          {{ row.description }}
        </NDescriptionsItem>
      </NDescriptions>

      <!-- Agent 状态 -->
      <div class="mb-16px flex items-center gap-8px">
        <span class="text-13px font-500">{{ $t('page.server.agentOps.agentSection') }}</span>
        <NDivider class="flex-1" />
      </div>
      <NDescriptions v-if="row" :column="2" label-placement="left" bordered size="small" class="mb-16px">
        <NDescriptionsItem :label="$t('page.server.asset.col.agentStatus')">
          <NTag size="small" :type="agentTagType" :bordered="false">{{ agentStatusLabel }}</NTag>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.agentOps.agentVersion')">{{ row.agentVersion || '-' }}</NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.agentOps.agentHostname')">{{ row.agentHostname || '-' }}</NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.server.agentOps.lastHeartbeat')">
          <NTime v-if="lastHeartbeatTime > 0" :time="lastHeartbeatTime" format="yyyy-MM-dd HH:mm:ss" />
          <span v-else>-</span>
        </NDescriptionsItem>
      </NDescriptions>

      <!-- 资源状态(agent 快照) -->
      <div class="mb-16px flex items-center gap-8px">
        <span class="text-13px font-500">{{ $t('page.server.agentOps.resourceSection') }}</span>
        <NDivider class="flex-1" />
      </div>
      <NSpin :show="snapshotLoading" size="small">
        <div v-if="snapshot && !snapshot.stale" class="grid grid-cols-2 gap-12px">
          <NCard size="small" class="card-wrapper">
            <div class="flex items-center justify-between">
              <span class="text-13px text-slate-400">CPU</span>
              <span class="text-18px font-600" :class="snapshot.cpuPercent > 85 ? 'color-error' : snapshot.cpuPercent > 70 ? 'color-warning' : ''">
                {{ snapshot.cpuPercent.toFixed(1) }}%
              </span>
            </div>
            <NProgress type="line" :percentage="snapshot.cpuPercent" :height="6" class="mt-4px" />
            <span class="text-12px text-slate-400">{{ $t('page.server.agentOps.loadavg1') }}: {{ snapshot.loadavg1.toFixed(2) }}</span>
          </NCard>
          <NCard size="small" class="card-wrapper">
            <div class="flex items-center justify-between">
              <span class="text-13px text-slate-400">{{ $t('page.server.agentOps.memory') }}</span>
              <span class="text-18px font-600" :class="memUsedPercent(snapshot) > 90 ? 'color-error' : memUsedPercent(snapshot) > 80 ? 'color-warning' : ''">
                {{ memUsedPercent(snapshot).toFixed(1) }}%
              </span>
            </div>
            <NProgress type="line" :percentage="memUsedPercent(snapshot)" :height="6" class="mt-4px" />
            <span class="text-12px text-slate-400">
              {{ fmtKB(snapshot.memTotalKb - snapshot.memAvailableKb) }} / {{ fmtKB(snapshot.memTotalKb) }}
            </span>
          </NCard>
          <NCard size="small" class="card-wrapper">
            <div class="flex items-center justify-between">
              <span class="text-13px text-slate-400">{{ $t('page.server.agentOps.disk') }}(/)</span>
              <span class="text-18px font-600" :class="diskUsedPercent(snapshot) > 90 ? 'color-error' : diskUsedPercent(snapshot) > 80 ? 'color-warning' : ''">
                {{ diskUsedPercent(snapshot).toFixed(1) }}%
              </span>
            </div>
            <NProgress type="line" :percentage="diskUsedPercent(snapshot)" :height="6" class="mt-4px" />
            <span class="text-12px text-slate-400">
              {{ fmtBytes(snapshot.diskUsedBytes) }} / {{ fmtBytes(snapshot.diskTotalBytes) }}
            </span>
          </NCard>
          <NCard size="small" class="card-wrapper">
            <div class="flex flex-col gap-4px">
              <div class="flex items-center justify-between">
                <span class="text-13px text-slate-400">{{ $t('page.server.agentOps.network') }}</span>
              </div>
              <span class="text-13px">↓ {{ fmtBps(snapshot.netInBps) }}</span>
              <span class="text-13px">↑ {{ fmtBps(snapshot.netOutBps) }}</span>
            </div>
          </NCard>
          <NCard size="small" class="card-wrapper">
            <div class="flex items-center justify-between">
              <span class="text-13px text-slate-400">{{ $t('page.server.agentOps.uptime') }}</span>
              <span class="text-14px font-500">{{ fmtUptime(snapshot.uptimeSec) }}</span>
            </div>
          </NCard>
        </div>
        <NEmpty v-else :description="$t('page.server.agentOps.noSnapshot')" class="py-24px" />
      </NSpin>

      <!-- 运维操作 -->
      <div class="mb-8px flex items-center gap-8px">
        <span class="text-13px font-500">{{ $t('page.server.agentOps.opsSection') }}</span>
        <NDivider class="flex-1" />
      </div>
      <div class="flex flex-col gap-8px">
        <NSpace>
          <NButton v-if="!agentInstalled" size="small" type="primary" :disabled="!isSSHAsset || opsRunning" @click="installModalVisible = true">
            {{ $t('page.server.agentInstall.action') }}
          </NButton>
          <NButton v-if="agentInstalled" size="small" type="primary" ghost :loading="opsRunning" @click="triggerOps('restart')">
            {{ $t('page.server.agentOps.restart') }}
          </NButton>
          <NButton v-if="agentInstalled" size="small" type="error" ghost :disabled="opsRunning" @click="triggerOps('uninstall')">
            {{ $t('page.server.agentOps.uninstall') }}
          </NButton>
        </NSpace>

        <!-- 运维任务进度(重启/卸载) -->
        <div v-if="opsStatus" class="flex flex-col gap-4px">
          <div class="flex items-center gap-8px">
            <NSpin v-if="opsStatus.status === 'running'" :size="14" />
            <NTag v-else :type="opsStatus.status === 'success' ? 'success' : 'error'" size="small">
              {{ $t(`page.server.agentInstall.status_${opsStatus.status}`) }}
            </NTag>
            <span class="text-13px">{{ opsStatus.step }}</span>
          </div>
          <span v-if="opsStatus.message" class="text-12px" :class="opsStatus.status === 'failed' ? 'color-error' : 'text-slate-400'">
            {{ opsStatus.message }}
          </span>
        </div>
      </div>

      <AgentInstallModal v-model:visible="installModalVisible" :row="row" @finished="handleInstallFinished" />
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped></style>
