<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useLoading } from '@sa/hooks';
import SettingMenu from './modules/setting-menu.vue';
import GeneralSetting from './modules/general-setting.vue';
import SecuritySetting from './modules/security-setting.vue';
import { useAuth } from '@/hooks/business/auth';
import { fetchGetSetting, fetchUpdateSetting } from '@/service/api/system/setting';
import { useAppStore } from '@/store/modules/app/index.js';
import { mergeConfig } from '@/utils/common';

defineOptions({ name: 'SystemSetting' });

const { t } = useI18n();
const { hasAuth } = useAuth();
const { loading, startLoading, endLoading } = useLoading();

type SettingKey = 'general' | 'security';

interface SettingMenuItem {
  key: SettingKey;
  label: string;
  desc: string;
  icon: string;
}

/** 常规配置默认值 */
const GENERAL_DEFAULTS: Api.System.GeneralSettingConfig = {
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
};

/** 安全配置默认值 */
const SECURITY_DEFAULTS: Api.System.SecuritySettingConfig = {
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
};

const activeKey = ref<SettingKey>('general');
const isMobile = computed(() => useAppStore().isMobile);

const config = ref<{
  general: Api.System.GeneralSettingConfig;
  security: Api.System.SecuritySettingConfig;
}>({
  general: { ...GENERAL_DEFAULTS },
  security: { ...SECURITY_DEFAULTS }
});

/** 菜单项统一定义，同时传给 setting-menu 和 currentTitle */
const menuItems: SettingMenuItem[] = [
  {
    key: 'general',
    label: t('page.system.setting.general'),
    desc: t('page.system.setting.generalDesc'),
    icon: 'mdi:cog-outline'
  },
  {
    key: 'security',
    label: t('page.system.setting.security'),
    desc: t('page.system.setting.securityDesc'),
    icon: 'mdi:shield-check-outline'
  }
];

const currentTitle = computed(() => menuItems.find(i => i.key === activeKey.value)?.label ?? '');

async function loadConfig() {
  const { data, error } = await fetchGetSetting();
  if (error) {
    window.$message?.error(t('page.system.setting.loadFail'));
    return;
  }
  config.value.general = mergeConfig(GENERAL_DEFAULTS, data?.general);
  config.value.security = mergeConfig(SECURITY_DEFAULTS, data?.security);
}

async function handleSave() {
  startLoading();
  const { error } = await fetchUpdateSetting({
    general: { ...config.value.general },
    security: { ...config.value.security }
  });
  if (error) {
    window.$message?.error(t('page.system.setting.saveFail'));
  } else {
    window.$message?.success(t('page.system.setting.saveSuccess'));
  }
  endLoading();
}

onMounted(() => {
  loadConfig();
});
</script>

<template>
  <div>
    <div class="h-full overflow-hidden">
      <!-- Mobile layout -->
      <template v-if="isMobile">
        <NCard :bordered="false" class="card-wrapper h-full">
          <NCollapse>
            <NCollapseItem name="menu" title="配置选项">
              <SettingMenu v-model:active-key="activeKey" :menu-items="menuItems" />
            </NCollapseItem>
          </NCollapse>
          <NDivider class="my-12px" />
          <div class="flex justify-between items-center mb-16px">
            <div class="text-16px font-600">{{ currentTitle }}</div>
            <NButton v-if="hasAuth('system:setting:save')" type="primary" :loading="loading" class="dark:text-white" @click="handleSave">保存</NButton>
          </div>
          <div class="overflow-auto setting-mobile-content">
            <GeneralSetting v-if="activeKey === 'general'" v-model:config="config.general" />
            <SecuritySetting v-else-if="activeKey === 'security'" v-model:config="config.security" />
          </div>
        </NCard>
      </template>
      <!-- Desktop layout -->
      <template v-else>
        <div class="h-full flex flex-row">
          <div class="h-full w-1/5 flex-none">
            <SettingMenu v-model:active-key="activeKey" :menu-items="menuItems" />
          </div>
          <NCard :bordered="false" class="card-wrapper ml-4 h-full flex-1 flex flex-col" :content-style="{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }">
            <template #header>
              <div class="flex items-center justify-between">
                <span class="text-16px font-600">{{ currentTitle }}</span>
                <NButton v-if="hasAuth('system:setting:save')" type="primary" :loading="loading" @click="handleSave">
                  {{ $t('page.system.setting.save') }}
                </NButton>
              </div>
            </template>
            <div class="setting-content flex-1 overflow-y-auto overflow-x-hidden pr-1">
              <GeneralSetting v-if="activeKey === 'general'" v-model:config="config.general" />
              <SecuritySetting v-else-if="activeKey === 'security'" v-model:config="config.security" />
            </div>
          </NCard>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.setting-mobile-content {
  height: calc(100% - 180px);
}
</style>
