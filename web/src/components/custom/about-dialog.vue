<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { fetchCheckUpdate, fetchGetVersion, fetchStartUpgrade } from '@/service/api/system';
import { formatDateTime } from '@/utils/format';
import { $t } from '@/locales';
import { useAuth } from '@/hooks/business/auth';
import UpgradeProgressDialog from './upgrade-progress-dialog.vue';

defineOptions({ name: 'AboutDialog' });

interface Props {
  visible: boolean;
}

interface Emits {
  (e: 'update:visible', v: boolean): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const { hasAuth } = useAuth();

const loading = ref(false);
const versionInfo = ref<Api.System.UpgradeVersionInfo | null>(null);

// 检查更新状态
const checking = ref(false);
const checkResult = ref<Api.System.UpgradeCheckResult | null>(null);
// 触发升级：accepted 后打开进度弹窗轮询
const upgrading = ref(false);
const progressVisible = ref(false);

// 每次打开重新拉取：在线升级完成后新后端一启动，此处展示即为新版本；
// 顺带检查更新（失败静默——发布服务器未配置/不可达时只展示版本信息）
watch(
  () => props.visible,
  async v => {
    if (!v) return;
    checkResult.value = null;
    loading.value = true;
    const { data, error } = await fetchGetVersion();
    loading.value = false;
    versionInfo.value = !error && data ? data : null;
    await handleCheckUpdate(true);
  }
);

/** 手动/静默检查更新；静默模式（打开弹窗自动查）只在有新版本时落地结果，不打扰 */
async function handleCheckUpdate(silent = false) {
  checking.value = true;
  const { data, error } = await fetchCheckUpdate();
  checking.value = false;
  if (error || !data) {
    if (!silent) window.$message?.error($t('upgrade.checkFail'));
    return;
  }
  checkResult.value = data;
  if (!silent && !data.hasUpdate) {
    window.$message?.info(data.message || $t('upgrade.alreadyLatest'));
  }
}

/** changelog 极简行级渲染：## → 小标题，- / * → 列表项，其余按段落（不值得为此引入 markdown 库） */
const changeLogLines = computed(() => {
  const raw = checkResult.value?.changeLog?.trim();
  if (!raw) return [];
  return raw.split('\n').map(line => {
    const heading = line.match(/^#{1,6}\s+(.*)$/);
    if (heading) return { type: 'heading' as const, text: heading[1] };
    const item = line.match(/^\s*[-*]\s+(.*)$/);
    if (item) return { type: 'item' as const, text: item[1] };
    return { type: 'text' as const, text: line.trim() };
  });
});

/** 立即升级：转发 updater，accepted 后进度弹窗接管；updater 拒绝（进行中/不可达）弹提示 */
async function handleStartUpgrade() {
  upgrading.value = true;
  const { data, error } = await fetchStartUpgrade();
  upgrading.value = false;
  if (error || !data) {
    window.$message?.error($t('upgrade.startFail'));
    return;
  }
  if (data.accepted) {
    progressVisible.value = true;
  } else {
    window.$message?.warning(data.message || $t('upgrade.startFail'));
  }
}

/** 升级结束：刷新版本展示（升级成功即新版本号） */
function handleUpgradeFinished(success: boolean) {
  if (!success) return;
  fetchGetVersion().then(({ data, error }) => {
    if (!error && data) {
      versionInfo.value = data;
      checkResult.value = null;
    }
  });
}

/** 构建时间格式化为本地易读形式；未知（裸构建）与解析失败统一显示 '-' */
function formatBuildTime(buildTime?: string) {
  if (!buildTime || buildTime === 'unknown') return '-';
  const formatted = formatDateTime(buildTime);
  return formatted === buildTime ? '-' : formatted;
}
</script>

<template>
  <NModal
    :show="visible"
    preset="card"
    :title="$t('common.about')"
    class="w-420px"
    @update:show="(v: boolean) => emit('update:visible', v)"
  >
    <NSpin :show="loading">
      <div class="flex flex-col items-center gap-4px py-8px">
        <SystemLogo class="size-48px text-primary" />
        <div class="mt-4px text-17px font-bold">{{ versionInfo?.appName ?? $t('system.title') }}</div>
        <div class="text-13px opacity-60">{{ versionInfo?.description }}</div>
      </div>
      <NDivider class="!my-12px" />
      <div class="flex flex-col gap-10px px-8px text-14px">
        <div class="flex items-center justify-between">
          <span class="opacity-60">{{ $t('common.version') }}</span>
          <span class="font-medium">{{ versionInfo?.version ?? '-' }}</span>
        </div>
        <div class="flex items-center justify-between">
          <span class="opacity-60">{{ $t('common.buildTime') }}</span>
          <span class="font-medium">{{ formatBuildTime(versionInfo?.buildTime) }}</span>
        </div>
      </div>
      <!-- 新版本信息（changelog 行级渲染） -->
      <template v-if="checkResult?.hasUpdate">
        <NDivider class="!my-12px" />
        <div class="mx-8px rounded-6px bg-primary-50 p-12px dark:bg-primary-900/30">
          <div class="mb-6px flex items-center gap-8px">
            <SvgIcon icon="ph:arrow-up-circle" class="text-18px text-primary" />
            <span class="text-14px font-600">{{ $t('upgrade.foundNewVersion') }}</span>
            <span class="text-14px font-600 text-primary">{{ checkResult.version }}</span>
            <span v-if="checkResult.releaseTime" class="ml-auto text-12px opacity-50">
              {{ formatDateTime(checkResult.releaseTime) }}
            </span>
          </div>
          <div v-if="changeLogLines.length" class="flex flex-col gap-2px text-13px">
            <template v-for="(line, i) in changeLogLines" :key="i">
              <div v-if="line.type === 'heading'" class="mt-4px font-600">{{ line.text }}</div>
              <div v-else-if="line.type === 'item'" class="flex gap-6px pl-6px">
                <span class="opacity-40">•</span>
                <span class="flex-1">{{ line.text }}</span>
              </div>
              <div v-else-if="line.text" class="opacity-80">{{ line.text }}</div>
            </template>
          </div>
        </div>
      </template>
    </NSpin>
    <template #footer>
      <div class="flex-center gap-12px">
        <NButton round :loading="checking" @click="handleCheckUpdate()">{{ $t('common.checkUpdate') }}</NButton>
        <!-- 立即升级：需 system:setting:upgrade 权限（与系统设置保存同级管理操作）；升级期间服务会短暂重启 -->
        <NTooltip v-if="checkResult?.hasUpdate" trigger="hover">
          <template #trigger>
            <NButton
              v-if="hasAuth('system:setting:upgrade')"
              round
              type="primary"
              :loading="upgrading"
              @click="handleStartUpgrade"
            >
              {{ $t('upgrade.startNow') }}
            </NButton>
          </template>
          {{ $t('upgrade.startNowTip') }}
        </NTooltip>
      </div>
    </template>
    <UpgradeProgressDialog v-model:visible="progressVisible" @finished="handleUpgradeFinished" />
  </NModal>
</template>
