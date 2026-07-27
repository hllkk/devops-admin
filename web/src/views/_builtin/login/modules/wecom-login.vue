<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import QRCode from 'qrcode';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/store/modules/auth';
import { useThemeStore } from '@/store/modules/theme';
import { useSystemStore } from '@/store/modules/system';
import { $t } from '@/locales';
import { fetchQrCodeStatus, fetchWecomQrCode, fetchWecomWebviewLogin } from '@/service/api';
import { isWecomWebview } from '@/utils/agent';

defineOptions({
  name: 'WecomLogin'
});

const router = useRouter();
const authStore = useAuthStore();
const themeStore = useThemeStore();
const systemStore = useSystemStore();

// WebView 检测(一次会话内不变)
const isWebview = isWecomWebview();
const webviewRedirecting = ref(false);

const oauthUrl = ref('');
const sceneId = ref('');
const loading = ref(false);
const expired = ref(false);
const countdown = ref(120);
const errorMessage = ref('');
const pollInterval = ref(3000);
const lastStatus = ref('');
const scanned = ref(false);

const qrCanvas = ref<HTMLCanvasElement | null>(null);

let pollTimer: ReturnType<typeof setTimeout> | null = null;
let countdownTimer: ReturnType<typeof setInterval> | null = null;

/** QR 码渲染选项(暗黑模式反色) */
function getQrOptions(): QRCode.QRCodeRenderersOptions {
  return themeStore.darkMode
    ? { width: 240, margin: 3, color: { dark: '#d1d5db', light: '#1f2937' } }
    : { width: 240, margin: 3, color: { dark: '#1f2937', light: '#ffffff' } };
}

/** 渲染 QR 码到 canvas */
async function renderQrCode() {
  if (!qrCanvas.value || !oauthUrl.value) return;
  try {
    await QRCode.toCanvas(qrCanvas.value, oauthUrl.value, getQrOptions());
  } catch (e) {
    console.error('QR 码渲染失败:', e);
  }
}

/** WebView 免登失败:重置跳转态并展示错误 */
function failWebviewLogin() {
  webviewRedirecting.value = false;
  errorMessage.value = $t('page.login.wecomLogin.qrCodeLoadFailed');
}

/** WebView 免登:企微客户端内自动跳转 OAuth 授权链接 */
async function initWebviewLogin() {
  webviewRedirecting.value = true;
  try {
    const { data, error } = await fetchWecomWebviewLogin();
    if (error || !data) {
      failWebviewLogin();
      return;
    }
    window.location.replace(data.oauthUrl);
  } catch {
    failWebviewLogin();
  }
}

async function loadQrCode() {
  loading.value = true;
  expired.value = false;
  errorMessage.value = '';
  lastStatus.value = '';
  scanned.value = false;
  pollInterval.value = 3000;

  stopPolling();
  stopCountdown();

  try {
    const { data, error } = await fetchWecomQrCode();
    if (error || !data) {
      errorMessage.value = $t('page.login.wecomLogin.qrCodeLoadFailed');
      return;
    }
    sceneId.value = data.sceneId;
    countdown.value = data.countdown || 120;
    oauthUrl.value = data.oauthUrl;
    startCountdown();
    scheduleNextPoll();
  } catch {
    errorMessage.value = $t('page.login.wecomLogin.qrCodeLoadFailed');
  } finally {
    loading.value = false;
  }
}

// setTimeout 链式轮询:首次 3s,扫码后切 1s,命中终态停止
function scheduleNextPoll() {
  stopPolling();
  pollTimer = setTimeout(async () => {
    if (!sceneId.value) return;
    const { data, error } = await fetchQrCodeStatus(sceneId.value);
    if (!error && data) {
      if (data.status === 'scanned' && lastStatus.value !== 'scanned') {
        pollInterval.value = 1000;
        scanned.value = true;
      }
      lastStatus.value = data.status;

      if (data.status === 'confirmed' && data.expiresAt) {
        stopCountdown();
        await authStore.wecomLogin(data.expiresAt);
        return;
      }
      if (data.status === 'expired') {
        stopCountdown();
        expired.value = true;
        return;
      }
      if (data.status === 'fail') {
        stopCountdown();
        errorMessage.value = data.errMsg || $t('page.login.wecomLogin.qrCodeLoadFailed');
        return;
      }
    }
    scheduleNextPoll();
  }, pollInterval.value);
}

function startCountdown() {
  countdownTimer = setInterval(() => {
    countdown.value -= 1;
    if (countdown.value <= 0) {
      stopCountdown();
      expired.value = true;
      stopPolling();
    }
  }, 1000);
}

function stopPolling() {
  if (pollTimer) {
    clearTimeout(pollTimer);
    pollTimer = null;
  }
}

function stopCountdown() {
  if (countdownTimer) {
    clearInterval(countdownTimer);
    countdownTimer = null;
  }
}

function goBack() {
  router.push({ name: 'login', params: { module: 'pwd-login' } });
}

function refreshQrCode() {
  loadQrCode();
}

// 监听 oauthUrl 变化(初次加载)与暗黑模式切换,nextTick 确保 DOM 就绪后渲染
watch(
  [oauthUrl, () => themeStore.darkMode],
  () => {
    nextTick(() => {
      renderQrCode();
    });
  }
);

onUnmounted(() => {
  stopPolling();
  stopCountdown();
});

if (isWebview && !!systemStore.setting?.wecomEnabled) {
  onMounted(initWebviewLogin);
} else {
  loadQrCode();
}
</script>

<template>
  <div class="h-full flex-col-center gap-24px">
    <!-- WebView 免登跳转中 -->
    <div v-if="webviewRedirecting" class="flex-col-center h-340px">
      <NSpin size="large" />
      <p class="mt-12px text-14px text-gray-400">{{ $t('page.login.wecomLogin.loading') }}</p>
    </div>

    <!-- 加载中 -->
    <div v-else-if="loading" class="flex-col-center h-340px">
      <NSpin size="large" />
      <p class="mt-12px text-14px text-gray-400">{{ $t('page.login.wecomLogin.loading') }}</p>
    </div>

    <!-- 错误提示 -->
    <div v-else-if="errorMessage" class="flex-col-center h-340px">
      <div class="text-48px text-red-400">
        <icon-mdi-alert-circle-outline />
      </div>
      <p class="mt-12px text-14px text-red-400">{{ errorMessage }}</p>
      <NButton type="primary" size="small" class="mt-16px" @click="refreshQrCode">
        {{ $t('page.login.wecomLogin.refresh') }}
      </NButton>
    </div>

    <!-- QR 码卡片 -->
    <div v-else class="flex-col-center">
      <div class="qr-card relative" :class="{ expired, scanned }">
        <div class="qr-card-accent" />

        <div class="qr-card-body">
          <canvas ref="qrCanvas" class="qr-canvas" />

          <!-- 已扫码遮罩 -->
          <div v-if="scanned && !expired" class="qr-overlay-scanned">
            <div class="flex-col-center gap-12px">
              <icon-mdi-check-circle class="text-48px text-green-400" />
              <span class="text-16px text-white font-medium">{{ $t('page.login.wecomLogin.scanned') }}</span>
            </div>
          </div>

          <!-- 过期遮罩 -->
          <div v-if="expired" class="qr-overlay-expired">
            <div class="flex-col-center gap-8px">
              <icon-mdi-refresh class="text-32px text-white" />
              <NButton type="primary" size="small" @click="refreshQrCode">
                {{ $t('page.login.wecomLogin.refresh') }}
              </NButton>
            </div>
          </div>
        </div>

        <div class="qr-card-footer">
          <template v-if="scanned && !expired">
            <div class="qr-footer-status">
              <icon-mdi-check-circle class="text-16px text-green-500" />
              <span class="qr-tip-scanned">{{ $t('page.login.wecomLogin.scannedConfirm') }}</span>
            </div>
          </template>
          <template v-else-if="expired">
            <p class="qr-tip-expired">{{ $t('page.login.wecomLogin.expired') }}</p>
          </template>
          <template v-else>
            <p class="qr-tip">{{ $t('page.login.wecomLogin.scanTip') }}</p>
            <p class="qr-app-name">{{ $t('page.login.wecomLogin.appName') }}</p>
          </template>
        </div>
      </div>

      <div v-if="!expired && !scanned" class="mt-8px text-12px text-gray-400">
        {{ $t('page.login.wecomLogin.countdown', { seconds: countdown }) }}
      </div>
    </div>

    <NButton v-if="!webviewRedirecting" quaternary size="small" @click="goBack">
      <template #icon>
        <icon-mdi-arrow-left />
      </template>
      {{ $t('page.login.wecomLogin.backToLogin') }}
    </NButton>
  </div>
</template>

<style scoped>
.qr-card {
  width: 280px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  overflow: hidden;
  transition: background 0.3s, box-shadow 0.3s;
}

:root.dark .qr-card {
  background: #1f2937;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

.qr-card-accent {
  height: 4px;
  background: linear-gradient(90deg, #2b7ef9, #5b9dfa);
}

.qr-card-body {
  position: relative;
  padding: 20px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.qr-canvas {
  display: block;
  border-radius: 8px;
}

/* 已扫码遮罩:绿色半透明 + 模糊 */
.qr-overlay-scanned {
  position: absolute;
  inset: 0;
  background: rgba(82, 196, 26, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  backdrop-filter: blur(6px);
  border-radius: 0 0 12px 12px;
  animation: fadeIn 0.3s ease;
}

/* 过期遮罩:灰黑半透明 + 模糊 */
.qr-overlay-expired {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  display: flex;
  align-items: center;
  justify-content: center;
  backdrop-filter: blur(6px);
  border-radius: 0 0 12px 12px;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.qr-card-footer {
  padding: 12px 20px 16px;
  text-align: center;
  border-top: 1px solid #f3f4f6;
}

:root.dark .qr-card-footer {
  border-top-color: #374151;
}

.qr-tip {
  font-size: 15px;
  color: #374151;
  line-height: 1.6;
  margin: 0;
}

:root.dark .qr-tip {
  color: #e5e7eb;
}

.qr-app-name {
  font-size: 13px;
  color: #6b7280;
  font-weight: 600;
  margin: 6px 0 0;
}

:root.dark .qr-app-name {
  color: #9ca3af;
}

.qr-footer-status {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.qr-tip-scanned {
  font-size: 15px;
  color: #22c55e;
  font-weight: 600;
  margin: 0;
}

:root.dark .qr-tip-scanned {
  color: #4ade80;
}

.qr-tip-expired {
  font-size: 15px;
  color: #f59e0b;
  font-weight: 600;
  margin: 0;
}

:root.dark .qr-tip-expired {
  color: #fbbf24;
}

/* 移动端响应式:缩小卡片 */
@media (max-width: 640px) {
  .qr-card {
    width: 240px;
  }

  .qr-card-body {
    padding: 16px;
  }
}
</style>
