<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useLoading } from '@sa/hooks';
import SettingMenu from './modules/setting-menu.vue';
import GeneralSetting from './modules/general-setting.vue';
import SecuritySetting from './modules/security-setting.vue';
import LdapSetting from './modules/ldap-setting.vue';
import { useAuth } from '@/hooks/business/auth';
import { fetchGetSetting, fetchUpdateSetting } from '@/service/api/system/setting';
import { useAppStore } from '@/store/modules/app/index.js';
import { mergeConfig } from '@/utils/common';

defineOptions({ name: 'SystemSetting' });

const { t } = useI18n();
const { hasAuth } = useAuth();
const { loading, startLoading, endLoading } = useLoading();

type SettingKey = 'general' | 'security' | 'ldap' | 'disk' | 'notify' | 'auth';

interface SettingMenuItem {
  key: SettingKey;
  label: string;
  desc: string;
  icon: string;
}

/** 常规配置默认值(系统信息 + 日志清理,字段均在 SysGeneralConfig) */
const GENERAL_DEFAULTS: Api.System.GeneralSettingConfig = {
  systemName: 'devops-admin',
  systemDescription: '企业运维管理平台',
  logoUrl: '',
  faviconUrl: '',
  loginLogRetentionDays: 90,
  operationLogRetentionDays: 90
};

/** 安全配置默认值(对齐后端 SysSecurityConfig 的 gorm default 与 DefaultSecurityConfig) */
const SECURITY_DEFAULTS: Api.System.SecuritySettingConfig = {
  // 验证码 Captcha*
  captchaEnabled: true,
  captchaType: 'click',
  captchaOpen: 0,
  captchaTimeout: 3600,
  captchaTolerance: 5,
  keyLong: 6,
  imgWidth: 240,
  imgHeight: 80,
  // 密码复杂度 Password*
  passwordMinLength: 8,
  passwordRequireUppercase: false,
  passwordRequireLowercase: false,
  passwordRequireDigit: false,
  passwordRequireSpecial: false,
  // 登录失败锁定 LoginFailLock*
  loginFailLockCount: 5,
  loginFailLockTime: 30,
  // 访问控制 IpValidation*
  ipValidationEnabled: false,
  ipValidationMode: 'blacklist',
  ipBlacklist: '',
  ipWhitelist: '',
  // 限流 Limit*
  limitEnable: false,
  limitWindow: 60,
  limitCount: 30,
  // 密码过期 PwdExpire*
  pwdExpireEnable: false,
  pwdExpireDays: 90
};

/** LDAP 配置默认值(对齐后端 DefaultLdapConfig) */
const LDAP_DEFAULTS: Api.System.LdapSettingConfig = {
  enabled: false,
  host: 'localhost',
  port: 389,
  useSSL: false,
  bindDN: '',
  bindPass: '',
  baseDN: '',
  filter: '(uid=%s)',
  attrUsername: 'uid',
  attrNickname: 'cn',
  attrEmail: 'mail',
  autoCreate: false
};

const activeKey = ref<SettingKey>('general');
const isMobile = computed(() => useAppStore().isMobile);

const config = ref<{
  general: Api.System.GeneralSettingConfig;
  security: Api.System.SecuritySettingConfig;
  ldap: Api.System.LdapSettingConfig;
}>({
  general: { ...GENERAL_DEFAULTS },
  security: { ...SECURITY_DEFAULTS },
  ldap: { ...LDAP_DEFAULTS }
});

/** 菜单项统一定义，同时传给 setting-menu 和 currentTitle。computed 确保切换语言时标签/描述联动更新 */
const menuItems = computed<SettingMenuItem[]>(() => [
  {
    key: 'general',
    label: t('page.system.setting.general'),
    desc: t('page.system.setting.generalDesc'),
    icon: 'fluent-emoji:pushpin'
  },
  {
    key: 'security',
    label: t('page.system.setting.security'),
    desc: t('page.system.setting.securityDesc'),
    icon: 'fluent-emoji:locked'
  },
  {
    key: 'ldap',
    label: t('page.system.setting.ldap'),
    desc: t('page.system.setting.ldapDesc'),
    icon: 'fluent-emoji-flat:globe-with-meridians'
  },
  {
    key: 'disk',
    label: t('page.system.setting.disk'),
    desc: t('page.system.setting.diskDesc'),
    icon: 'fluent-emoji-flat:floppy-disk'
  },
  {
    key: 'notify',
    label: t('page.system.setting.notify'),
    desc: t('page.system.setting.notifyDesc'),
    icon: 'fluent-emoji-flat:loudspeaker'
  },
  {
    key: 'auth',
    label: t('page.system.setting.auth'),
    desc: t('page.system.setting.authDesc'),
    icon: 'fluent-emoji-flat:key'
  }
]);

const currentTitle = computed(() => menuItems.value.find(i => i.key === activeKey.value)?.label ?? '');

async function loadConfig() {
  const { data, error } = await fetchGetSetting();
  if (error) {
    window.$message?.error(t('page.system.setting.loadFail'));
    return;
  }
  config.value.general = mergeConfig(GENERAL_DEFAULTS, data?.general);
  config.value.security = mergeConfig(SECURITY_DEFAULTS, data?.security);
  config.value.ldap = mergeConfig(LDAP_DEFAULTS, data?.ldap);
}

async function handleSave() {
  startLoading();
  const { error } = await fetchUpdateSetting({
    general: { ...config.value.general },
    security: { ...config.value.security },
    ldap: { ...config.value.ldap }
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
          <SecuritySetting v-else-if="activeKey === 'security'" v-model:security-config="config.security" />
          <LdapSetting v-else-if="activeKey === 'ldap'" v-model:config="config.ldap" />
        </div>
      </NCard>
    </template>
    <!-- Desktop layout -->
    <template v-else>
      <div class="h-full flex flex-row">
        <div class="h-full w-1/5 flex-none min-h-0 overflow-y-auto">
          <SettingMenu v-model:active-key="activeKey" :menu-items="menuItems" />
        </div>
        <NCard :bordered="false" class="card-wrapper ml-4 h-full flex-1 flex flex-col min-h-0" content-scrollable>
          <template #header>
            <div class="flex items-center justify-between">
              <span class="text-16px font-600">{{ currentTitle }}</span>
              <NButton v-if="hasAuth('system:setting:save')" type="primary" :loading="loading" @click="handleSave">
                {{ $t('page.system.setting.save') }}
              </NButton>
            </div>
          </template>
          <div class="flex-1 min-h-0 overflow-y-auto overflow-x-hidden pr-1">
            <GeneralSetting v-if="activeKey === 'general'" v-model:config="config.general" />
            <SecuritySetting v-else-if="activeKey === 'security'" v-model:security-config="config.security" />
            <LdapSetting v-else-if="activeKey === 'ldap'" v-model:config="config.ldap" />
          </div>
        </NCard>
      </div>
    </template>
  </div>
</template>

<style scoped>
.setting-mobile-content {
  height: calc(100% - 180px);
}
</style>
