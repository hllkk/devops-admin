<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import SettingMenu from './modules/setting-menu.vue';
import GeneralSetting from './modules/general-setting.vue';
import SecuritySetting from './modules/security-setting.vue';
import { useAuth } from '@/hooks/business/auth';
import { fetchGetSetting, fetchUpdateSetting } from '@/service/api/system/setting';

defineOptions({ name: 'SystemSetting' });

const { t } = useI18n();
const { hasAuth } = useAuth();

type SettingKey = 'general' | 'security';

interface SettingForm {
  general: Api.System.GeneralSettingConfig;
  security: Api.System.SecuritySettingConfig;
}

const loading = ref(false);
const activeKey = ref<SettingKey>('general');

const config = ref<SettingForm>({
  general: {
    systemName: 'devops-admin',
    systemDescription: '企业运维管理平台',
    logoUrl: '',
    faviconUrl: '',
    userDefaultPassword: '',
    userDefaultRole: null,
    enableVerifyCode: false,
    verifyCodeType: 'click',
    verifyCodeLen: 4,
    verifyCodeExp: 5,
    verifyCodeTokenExp: 5,
    verifyInaccuracy: 40,
    loginLogRetentionDays: 90,
    operationLogRetentionDays: 90
  },
  security: {
    passwordMinLength: 8,
    passwordRequireUppercase: false,
    passwordRequireLowercase: true,
    passwordRequireDigit: true,
    passwordRequireSpecial: true,
    loginFailLockCount: 5,
    loginFailLockTime: 30,
    ipValidationEnabled: false,
    ipValidationMode: 'blacklist',
    ipBlacklist: '',
    ipWhitelist: ''
  }
});

const menuList = computed(() => [
  { key: 'general' as SettingKey, title: t('page.system.setting.general') },
  { key: 'security' as SettingKey, title: t('page.system.setting.security') }
]);

const currentTitle = computed(() => menuList.value.find(i => i.key === activeKey.value)?.title ?? '');

async function loadConfig() {
  const { data, error } = await fetchGetSetting();
  if (error) {
    window.$message?.error(t('page.system.setting.loadFail'));
    return;
  }
  if (data?.general) {
    const g = data.general;
    config.value.general = {
      systemName: g.systemName ?? '',
      systemDescription: g.systemDescription ?? '',
      logoUrl: g.logoUrl ?? '',
      faviconUrl: g.faviconUrl ?? '',
      userDefaultPassword: g.userDefaultPassword ?? '',
      userDefaultRole: g.userDefaultRole ?? null,
      enableVerifyCode: g.enableVerifyCode ?? false,
      verifyCodeType: g.verifyCodeType ?? 'click',
      verifyCodeLen: g.verifyCodeLen ?? 4,
      verifyCodeExp: g.verifyCodeExp ?? 5,
      verifyCodeTokenExp: g.verifyCodeTokenExp ?? 5,
      verifyInaccuracy: g.verifyInaccuracy ?? 40,
      loginLogRetentionDays: g.loginLogRetentionDays ?? 90,
      operationLogRetentionDays: g.operationLogRetentionDays ?? 90
    };
  }
  if (data?.security) {
    const s = data.security;
    config.value.security = {
      passwordMinLength: s.passwordMinLength ?? 8,
      passwordRequireUppercase: s.passwordRequireUppercase ?? false,
      passwordRequireLowercase: s.passwordRequireLowercase ?? true,
      passwordRequireDigit: s.passwordRequireDigit ?? true,
      passwordRequireSpecial: s.passwordRequireSpecial ?? true,
      loginFailLockCount: s.loginFailLockCount ?? 5,
      loginFailLockTime: s.loginFailLockTime ?? 30,
      ipValidationEnabled: s.ipValidationEnabled ?? false,
      ipValidationMode: s.ipValidationMode ?? 'blacklist',
      ipBlacklist: s.ipBlacklist ?? '',
      ipWhitelist: s.ipWhitelist ?? ''
    };
  }
}

async function handleSave() {
  loading.value = true;
  const { error } = await fetchUpdateSetting({
    general: { ...config.value.general },
    security: { ...config.value.security }
  });
  if (error) {
    window.$message?.error(t('page.system.setting.saveFail'));
  } else {
    window.$message?.success(t('page.system.setting.saveSuccess'));
  }
  loading.value = false;
}

onMounted(() => {
  loadConfig();
});
</script>

<template>
  <div class="h-full flex flex-col gap-12px lg:flex-row">
    <!-- 左侧菜单 -->
    <div class="h-full flex-none lg:w-220px">
      <NCard :bordered="false" class="h-full card-wrapper">
        <SettingMenu v-model:active-key="activeKey" />
      </NCard>
    </div>

    <!-- 右侧内容 -->
    <NCard :bordered="false" class="h-full flex-1 card-wrapper">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="text-16px font-600">{{ currentTitle }}</span>
          <NButton v-if="hasAuth('system:setting:save')" type="primary" :loading="loading" @click="handleSave">
            {{ $t('page.system.setting.save') }}
          </NButton>
        </div>
      </template>
      <div class="overflow-auto pr-8px">
        <GeneralSetting v-if="activeKey === 'general'" v-model:config="config.general" />
        <SecuritySetting v-else-if="activeKey === 'security'" v-model:config="config.security" />
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.card-wrapper {
  border-radius: 8px;
}
</style>
