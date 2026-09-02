<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useLoading } from '@sa/hooks';
import SettingMenu from './modules/setting-menu.vue';
import GeneralSetting from './modules/general-setting.vue';
import SecuritySetting from './modules/security-setting.vue';
import LdapSetting from './modules/ldap-setting.vue';
import NotifySetting from './modules/notify-setting.vue';
import AuthSetting from './modules/auth-setting.vue';
import { useAuth } from '@/hooks/business/auth';
import { fetchGetSetting, fetchUpdateSetting } from '@/service/api/system/setting';
import { useAppStore } from '@/store/modules/app/index.js';

defineOptions({ name: 'SystemSetting' });

const { t } = useI18n();
const { hasAuth } = useAuth();
const { loading, startLoading, endLoading } = useLoading();

type SettingKey = 'general' | 'security' | 'ldap' | 'notify' | 'auth';

interface SettingMenuItem {
  key: SettingKey;
  label: string;
  desc: string;
  icon: string;
}

// 系统设置默认值由后端在初始化(/initdb)时 seed(见 source/system/sys_setting.go),
// 前端不再维护默认值副本: 此前 XXX_DEFAULTS 与后端 DefaultXxxConfig 已出现不一致
// (loginLogRetentionDays 90 vs 30、systemName 'devops-admin' vs 'DevOps Admin' 等),
// 现以后端为唯一真源, loadConfig 直接采用后端返回值。

const activeKey = ref<SettingKey>('general');
const loaded = ref(false);
const isMobile = computed(() => useAppStore().isMobile);

const config = ref<{
  general: Api.System.GeneralSettingConfig;
  security: Api.System.SecuritySettingConfig;
  ldap: Api.System.LdapSettingConfig;
  notify: Api.System.NotifySettingConfig;
  notifyPolicy: Api.System.NotifyPolicyConfig;
  auth: Api.System.AuthSettingConfig;
}>({
  general: {} as Api.System.GeneralSettingConfig,
  security: {} as Api.System.SecuritySettingConfig,
  ldap: {} as Api.System.LdapSettingConfig,
  notify: {} as Api.System.NotifySettingConfig,
  notifyPolicy: {
    sceneKey: 'token_plan_morning',
    enabled: false,
    targetType: 'users',
    targetIds: [],
    sendTime: '08:33',
    params: {}
  },
  auth: {} as Api.System.AuthSettingConfig
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
  config.value.general = data?.general ?? config.value.general;
  config.value.security = data?.security ?? config.value.security;
  config.value.ldap = data?.ldap ?? config.value.ldap;
  config.value.notify = data?.notify ?? config.value.notify;
  config.value.notifyPolicy = {
    ...config.value.notifyPolicy,
    ...data?.notifyPolicy,
    targetIds: data?.notifyPolicy?.targetIds ?? [],
    sendTime: data?.notifyPolicy?.sendTime || '08:33',
    params: { ...data?.notifyPolicy?.params }
  };
  config.value.auth = data?.auth ?? config.value.auth;
  loaded.value = true;
}

async function handleSave() {
  startLoading();
  const { error } = await fetchUpdateSetting({
    general: { ...config.value.general },
    security: { ...config.value.security },
    ldap: { ...config.value.ldap },
    notify: { ...config.value.notify },
    notifyPolicy: { ...config.value.notifyPolicy, targetIds: [...config.value.notifyPolicy.targetIds] },
    auth: { ...config.value.auth }
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
          <NButton
            v-if="hasAuth('system:setting:save')"
            type="primary"
            :loading="loading"
            :disabled="!loaded"
            class="dark:text-white"
            @click="handleSave"
          >
            保存
          </NButton>
        </div>
        <div class="overflow-auto setting-mobile-content">
          <GeneralSetting v-if="activeKey === 'general'" v-model:config="config.general" />
          <SecuritySetting v-else-if="activeKey === 'security'" v-model:security-config="config.security" />
          <LdapSetting v-else-if="activeKey === 'ldap'" v-model:config="config.ldap" />
          <NotifySetting v-else-if="activeKey === 'notify'" v-model:config="config.notify" v-model:policy="config.notifyPolicy" />
          <AuthSetting v-else-if="activeKey === 'auth'" v-model:config="config.auth" />
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
              <NButton
                v-if="hasAuth('system:setting:save')"
                type="primary"
                :loading="loading"
                :disabled="!loaded"
                @click="handleSave"
              >
                {{ $t('page.system.setting.save') }}
              </NButton>
            </div>
          </template>
          <div class="flex-1 min-h-0 overflow-y-auto overflow-x-hidden pr-1">
            <GeneralSetting v-if="activeKey === 'general'" v-model:config="config.general" />
            <SecuritySetting v-else-if="activeKey === 'security'" v-model:security-config="config.security" />
            <LdapSetting v-else-if="activeKey === 'ldap'" v-model:config="config.ldap" />
            <NotifySetting v-else-if="activeKey === 'notify'" v-model:config="config.notify" v-model:policy="config.notifyPolicy" />
            <AuthSetting v-else-if="activeKey === 'auth'" v-model:config="config.auth" />
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
