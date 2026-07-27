<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { useLoading } from '@sa/hooks';
import CryptoJS from 'crypto-js';
import { Click as GoCaptchaClick, Rotate as GoCaptchaRotate, Slide as GoCaptchaSlide } from 'go-captcha-vue';
import 'go-captcha-vue/dist/style.css';
import { fetchCaptcha } from '@/service/api';
import { fetchSocialAuthBinding } from '@/service/api/system';
import { useAuthStore } from '@/store/modules/auth';
import { useSystemStore } from '@/store/modules/system';
import { useRouterPush } from '@/hooks/common/router';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { localStg } from '@/utils/storage';
import { decryptWithAes, encryptWithAes } from '@/utils/crypto';
import { $t } from '@/locales';

const aesKey = CryptoJS.enc.Utf8.parse(import.meta.env.VITE_REMEMBER_ME_AES_KEY || 'pC4aO6cD2uU7hA0bK6iD4vE1mV8sU8xG');

defineOptions({
  name: 'PwdLogin'
});

const authStore = useAuthStore();
const systemStore = useSystemStore();
const { toggleLoginModule } = useRouterPush();
const { formRef, validate } = useNaiveForm();
const { loading: captchaLoading, startLoading: startCaptchaLoading, endLoading: endCaptchaLoading } = useLoading();

const remberMe = ref<boolean>(false);

// 仅开发模式预填初始账号便于联调；生产构建（import.meta.env.DEV=false）经死代码消除后为空，避免登录页预填弱口令
const model: Api.Auth.PwdLoginForm = reactive({

});

// ===== go-captcha 行为验证码状态 =====
const captchaEnabled = ref<boolean>(false); // 当前是否要求验证码（后端触发策略决定）
const captchaType = ref<string>(''); // click | slide | rotate
const captchaData = ref<Api.Auth.CaptchaResult>({ captchaEnabled: false });
const captchaVerified = ref<boolean>(false); // 用户是否已通过本次验证码
const showCaptcha = ref<boolean>(false); // 行为验证码弹窗（click/slide/rotate）
// image 传统图形验证码：用户输入的图中字符（随登录一并提交，不走弹窗）
const imageCaptcha = ref<string>('');
// 当前行为验证码的用户答案（confirm 回调写入，提交登录时读取），无需响应式
let captchaAnswer = '';

type RuleKey = Extract<keyof Api.Auth.PwdLoginForm, 'username' | 'password'>;

const rules = computed<Record<RuleKey, App.Global.FormRule[]>>(() => {
  const { createRequiredRule } = useFormRules();
  return {
    username: [createRequiredRule($t('form.userName.required'))],
    password: [createRequiredRule($t('form.pwd.required'))]
  };
});

const captchaTitle = computed(() => {
  switch (captchaType.value) {
    case 'click':
      return $t('page.login.captcha.clickTitle');
    case 'slide':
      return $t('page.login.captcha.slideTitle');
    case 'rotate':
      return $t('page.login.captcha.rotateTitle');
    default:
      return $t('page.login.captcha.title');
  }
});

/** 拉取验证码：后端按触发策略返回，captchaEnabled=false 时无需验证 */
async function getCaptcha(username?: string) {
  startCaptchaLoading();
  const { data } = await fetchCaptcha(username);
  endCaptchaLoading();
  captchaEnabled.value = data?.captchaEnabled ?? false;
  captchaVerified.value = false;
  captchaAnswer = '';
  imageCaptcha.value = '';
  if (captchaEnabled.value && data) {
    captchaType.value = data.type ?? '';
    captchaData.value = data;
  }
}

/** 提交登录，携带验证码会话 ID 与用户答案 */
async function doLogin() {
  model.captchaId = captchaData.value.captchaId;
  // image=用户输入的图中字符；click/slide/rotate=confirm 回调序列化的坐标/角度
  model.captcha = captchaType.value === 'image' ? imageCaptcha.value : captchaVerified.value ? captchaAnswer : '';
  try {
    await authStore.login(model);
  } catch {
    // 登录失败：重置验证态并刷新（连续失败可能触发验证码）
    captchaVerified.value = false;
    captchaAnswer = '';
    await getCaptcha(model.username);
  }
}

async function handleSubmit() {
  await validate();
  // 记住密码
  if (remberMe.value) {
    const { username, password } = model;
    localStg.set('loginRember', encryptWithAes(JSON.stringify({ username, password }), aesKey));
  } else {
    localStg.remove('loginRember');
  }
  // 未验证时刷新验证码状态（带 username，确保阈值判断准确）
  // image 走内联输入，无"已验证"概念，保留当前验证码图不重复拉取
  if (!captchaVerified.value && captchaType.value !== 'image') {
    await getCaptcha(model.username);
  }
  // image 传统图形验证码：校验非空后随登录一并提交，不开行为验证弹窗
  if (captchaEnabled.value && captchaType.value === 'image') {
    if (!imageCaptcha.value.trim()) {
      window.$message?.warning?.($t('page.login.captcha.imageRequired'));
      return;
    }
    await doLogin();
    return;
  }
  if (captchaEnabled.value && !captchaVerified.value) {
    showCaptcha.value = true;
    return;
  }
  await doLogin();
}

// ===== go-captcha 组件回调 =====
// confirm 序列化用户答案，标记已验证，关闭弹窗并触发登录
function onConfirmClick(dots: { x: number; y: number }[]) {
  captchaAnswer = JSON.stringify(dots.map(d => ({ x: d.x, y: d.y })));
  finishCaptcha();
  return true;
}
function onConfirmSlide(point: { x: number; y: number }) {
  captchaAnswer = JSON.stringify({ x: point.x, y: point.y });
  finishCaptcha();
  return true;
}
function onConfirmRotate(angle: number) {
  captchaAnswer = JSON.stringify({ angle });
  finishCaptcha();
  return true;
}
function finishCaptcha() {
  captchaVerified.value = true;
  showCaptcha.value = false;
  doLogin();
}
async function onCaptchaRefresh() {
  await getCaptcha(model.username);
}

// 初始探测验证码状态（仅按 IP 判断，用户名待提交时再带上）
getCaptcha();

function handleLoginRember() {
  const loginRember = localStg.get('loginRember');
  if (!loginRember) return;
  try {
    remberMe.value = true;
    Object.assign(model, JSON.parse(decryptWithAes(loginRember, aesKey)));
  } catch {}
}
handleLoginRember();

async function handleSocialLogin(type: Api.System.SocialSource) {
  const { data, error } = await fetchSocialAuthBinding(type);
  if (error) return;
  window.location.href = data;
}
</script>

<template>
  <div class="mt-25">
    <div class="mb-5px text-32px text-black font-600 dark:text-white">登录到您的账户</div>
    <div class="pb-18px text-16px text-#858585">欢迎回来！请输入您的账户信息</div>
    <NForm
      ref="formRef"
      :model="model"
      :rules="rules"
      size="large"
      :show-label="false"
      @keyup.enter="() => !authStore.loginLoading && handleSubmit()"
    >
      <NFormItem path="username">
        <NInput v-model:value="model.username" :placeholder="$t('page.login.common.userNamePlaceholder')" />
      </NFormItem>
      <NFormItem path="password">
        <NInput
          v-model:value="model.password"
          type="password"
          show-password-on="click"
          :placeholder="$t('page.login.common.passwordPlaceholder')"
        />
      </NFormItem>
      <!-- image 传统字符图片验证码：内联输入框 + 可点击刷新的图片 -->
      <NFormItem v-if="captchaEnabled && captchaType === 'image'">
        <div class="flex w-full items-center gap-12px">
          <NInput
            v-model:value="imageCaptcha"
            class="flex-1"
            :placeholder="$t('page.login.captcha.imagePlaceholder')"
            :maxlength="10"
            @keyup.enter="() => !authStore.loginLoading && handleSubmit()"
          />
          <img
            v-if="captchaData.masterImage"
            :src="captchaData.masterImage"
            class="h-42px w-auto cursor-pointer select-none rounded-4px border border-solid border-#e0e0e6 dark:border-#2d2d33"
            :alt="$t('page.login.captcha.title')"
            :title="$t('page.login.captcha.refresh')"
            @click="onCaptchaRefresh"
          />
        </div>
      </NFormItem>
      <NSpace vertical :size="12" class="mb-8px">
        <div class="mx-6px mb-8px flex-y-center justify-between">
          <NCheckbox v-model:checked="remberMe" size="large">{{ $t('page.login.pwdLogin.rememberMe') }}</NCheckbox>
          <NA v-if="systemStore.isResetPwdEnabled" type="primary" class="text-18px" @click="toggleLoginModule('reset-pwd')">
            {{ $t('page.login.pwdLogin.forgetPassword') }}
          </NA>
          <span v-else class="text-18px" /> <!-- 占位保持布局对齐 -->
        </div>
        <NButton type="primary" size="large" block :loading="authStore.loginLoading" @click="handleSubmit">
          {{ captchaEnabled && captchaType !== 'image' && !captchaVerified ? $t('page.login.captcha.loginWithCaptcha') : $t('common.login') }}
        </NButton>
        <NButton v-if="systemStore.isRegisterEnabled" size="large" block @click="toggleLoginModule('register')">
          {{ $t('page.login.common.register') }}
        </NButton>
      </NSpace>
    </NForm>

    <NDivider v-if="systemStore.hasAnyThirdPartyLogin">
      <div class="color-#858585">{{ $t('page.login.pwdLogin.otherAccountLogin') }}</div>
    </NDivider>

    <div v-if="systemStore.hasAnyThirdPartyLogin" class="w-full flex flex-wrap justify-center gap-16px">
      <NButton v-if="systemStore.setting?.wecomEnabled" class="flex-1 min-w-100px" @click="toggleLoginModule('wecom-login')">
        <template #icon>
          <icon-tdesign-logo-wecom class="color-#2B7EF9" />
        </template>
        <span class="ml-6px">WeCom</span>
      </NButton>
      <NButton v-if="systemStore.setting?.wechatEnabled" class="flex-1 min-w-100px" @click="handleSocialLogin('wechat_open')">
        <template #icon>
          <icon-mdi-wechat class="color-#07C160" />
        </template>
        <span class="ml-6px">WeChat</span>
      </NButton>
      <NButton v-if="systemStore.setting?.giteeEnabled" class="flex-1 min-w-100px" @click="handleSocialLogin('gitee')">
        <template #icon>
          <icon-simple-icons-gitee class="color-#c71d23" />
        </template>
        <span class="ml-6px">Gitee</span>
      </NButton>
      <NButton v-if="systemStore.setting?.githubEnabled" class="flex-1 min-w-100px" @click="handleSocialLogin('github')">
        <template #icon>
          <icon-mdi-github class="color-#010409 dark:color-white" />
        </template>
        <span class="ml-6px">GitHub</span>
      </NButton>
    </div>

    <!-- 行为验证码弹窗（按 config.type 动态渲染） -->
    <NModal v-model:show="showCaptcha" :mask-closable="false" :close-on-esc="false">
      <NCard :title="captchaTitle" style="width: 400px" :bordered="false" role="dialog">
        <NSpin :show="captchaLoading">
          <GoCaptchaClick
            v-if="captchaType === 'click'"
            :config="{ title: $t('page.login.captcha.clickTitle') }"
            :data="{ image: captchaData.masterImage ?? '', thumb: captchaData.thumbImage ?? '' }"
            :events="{ confirm: onConfirmClick, refresh: onCaptchaRefresh }"
          />
          <GoCaptchaSlide
            v-else-if="captchaType === 'slide'"
            :config="{ title: $t('page.login.captcha.slideTitle') }"
            :data="{
              image: captchaData.masterImage ?? '',
              thumb: captchaData.tileImage ?? '',
              thumbX: captchaData.thumbX ?? 0,
              thumbY: captchaData.thumbY ?? 0,
              thumbWidth: captchaData.thumbWidth ?? 0,
              thumbHeight: captchaData.thumbHeight ?? 0
            }"
            :events="{ confirm: onConfirmSlide, refresh: onCaptchaRefresh }"
          />
          <GoCaptchaRotate
            v-else-if="captchaType === 'rotate'"
            :config="{ title: $t('page.login.captcha.rotateTitle') }"
            :data="{
              image: captchaData.masterImage ?? '',
              thumb: captchaData.thumbImage ?? '',
              angle: captchaData.angle ?? 0,
              thumbSize: captchaData.thumbSize ?? 0
            }"
            :events="{ confirm: onConfirmRotate, refresh: onCaptchaRefresh }"
          />
        </NSpin>
      </NCard>
    </NModal>
  </div>
</template>

<style scoped>
:deep(.n-base-selection),
:deep(.n-input) {
  --n-height: 42px !important;
  --n-font-size: 16px !important;
  --n-border-radius: 8px !important;
}

:deep(.n-base-selection-label) {
  padding: 0 6px !important;
}

:deep(.n-checkbox) {
  --n-size: 18px !important;
  --n-font-size: 16px !important;
}

:deep(.n-button) {
  --n-height: 42px !important;
  --n-font-size: 18px !important;
  --n-border-radius: 8px !important;
}

:deep(.n-divider) {
  --n-font-size: 16px !important;
  --n-font-weight: 400 !important;
}
</style>
