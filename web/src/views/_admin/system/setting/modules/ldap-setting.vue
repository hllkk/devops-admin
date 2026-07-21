<script setup lang="ts">
import { useI18n } from 'vue-i18n';

defineOptions({ name: 'LdapSetting' });

const { t } = useI18n();

const configModel = defineModel<Api.System.LdapSettingConfig>('config', { required: true });

const filterExamples = [
  { label: 'OpenLDAP: (uid=%s)', value: '(uid=%s)' },
  { label: 'AD: (sAMAccountName=%s)', value: '(sAMAccountName=%s)' },
  { label: 'AD-UPN: (userPrincipalName=%s)', value: '(userPrincipalName=%s)' },
  { label: 'Email: (mail=%s)', value: '(mail=%s)' }
];

function handleTestConnection() {
  window.$message?.info('测试连接功能开发中...');
}
</script>

<template>
  <NTabs type="line" animated>
    <!-- 连接配置 -->
    <NTabPane name="connection" :tab="t('page.system.setting.tabLdapConnection')">
      <NForm :model="configModel" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.ldapEnabled')" path="enabled">
          <NSwitch v-model:value="configModel.enabled" />
        </NFormItem>
        <template v-if="configModel.enabled">
          <NFormItem :label="$t('page.system.setting.ldapHost')" path="host">
            <NInput
              v-model:value="configModel.host"
              :placeholder="$t('page.system.setting.ldapHostPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.ldapPort')" path="port">
            <NInputNumber v-model:value="configModel.port" :min="1" :max="65535" class="max-w-200px" />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.ldapUseSSL')" path="useSSL">
            <NSwitch v-model:value="configModel.useSSL" />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.ldapBindDN')" path="bindDN">
            <NInput
              v-model:value="configModel.bindDN"
              :placeholder="$t('page.system.setting.ldapBindDNPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.ldapBindPass')" path="bindPass">
            <NInput
              v-model:value="configModel.bindPass"
              type="password"
              show-password-on="click"
              :placeholder="$t('page.system.setting.ldapBindPassPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.ldapBaseDN')" path="baseDN">
            <NInput
              v-model:value="configModel.baseDN"
              :placeholder="$t('page.system.setting.ldapBaseDNPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.ldapFilter')" path="filter">
            <div class="flex flex-col gap-8px max-w-400px">
              <NSelect
                v-model:value="configModel.filter"
                :options="filterExamples"
                :placeholder="$t('page.system.setting.ldapFilterPlaceholder')"
                clearable
                tag
                filterable
              />
            </div>
          </NFormItem>
          <NFormItem :label-width="160">
            <NButton type="success" @click="handleTestConnection">
              {{ $t('page.system.setting.ldapTestConnection') }}
            </NButton>
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>

    <!-- 属性映射 -->
    <NTabPane name="attrMap" :tab="t('page.system.setting.tabLdapAttrMap')">
      <NForm :model="configModel" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.ldapAttrUsername')" path="attrUsername">
          <NInput v-model:value="configModel.attrUsername" class="max-w-200px" />
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.ldapAttrNickname')" path="attrNickname">
          <NInput v-model:value="configModel.attrNickname" class="max-w-200px" />
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.ldapAttrEmail')" path="attrEmail">
          <NInput v-model:value="configModel.attrEmail" class="max-w-200px" />
        </NFormItem>
      </NForm>
    </NTabPane>

    <!-- 用户策略 -->
    <NTabPane name="userPolicy" :tab="t('page.system.setting.tabLdapUserPolicy')">
      <NForm :model="configModel" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.ldapAutoCreate')" path="autoCreate">
          <NSwitch v-model:value="configModel.autoCreate" />
          <span class="ml-12px text-12px text-gray-400">{{ $t('page.system.setting.ldapAutoCreateTip') }}</span>
        </NFormItem>
      </NForm>
    </NTabPane>
  </NTabs>
</template>
