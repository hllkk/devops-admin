<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

defineOptions({ name: 'GeneralSetting' });

const { t } = useI18n();

const config = defineModel<Api.System.GeneralSettingConfig>('config', { required: true });

const captchaTypeOptions = computed(() => [
  { label: t('page.system.setting.captchaType.click'), value: 'click' },
  { label: t('page.system.setting.captchaType.slide'), value: 'slide' },
  { label: t('page.system.setting.captchaType.dragdrop'), value: 'dragdrop' },
  { label: t('page.system.setting.captchaType.rotate'), value: 'rotate' }
]);
</script>

<template>
  <NForm :model="config" label-placement="left" :label-width="160">
    <NFormItem :label="$t('page.system.setting.systemName')" path="systemName">
      <NInput v-model:value="config.systemName" :placeholder="$t('page.system.setting.systemName')" class="max-w-400px" />
    </NFormItem>
    <NFormItem :label="$t('page.system.setting.systemDescription')" path="systemDescription">
      <NInput v-model:value="config.systemDescription" :placeholder="$t('page.system.setting.systemDescription')" class="max-w-400px" />
    </NFormItem>
    <NFormItem :label="$t('page.system.setting.logoUrl')" path="logoUrl">
      <div class="flex items-center gap-12px">
        <NInput v-model:value="config.logoUrl" :placeholder="$t('page.system.setting.logoUrl')" class="max-w-360px" />
        <img v-if="config.logoUrl" :src="config.logoUrl" alt="logo" class="max-h-40px max-w-120px rounded-4px" />
      </div>
    </NFormItem>
    <NFormItem :label="$t('page.system.setting.faviconUrl')" path="faviconUrl">
      <div class="flex items-center gap-12px">
        <NInput v-model:value="config.faviconUrl" :placeholder="$t('page.system.setting.faviconUrl')" class="max-w-360px" />
        <img v-if="config.faviconUrl" :src="config.faviconUrl" alt="favicon" class="size-32px rounded-4px" />
      </div>
    </NFormItem>
    <NFormItem :label="$t('page.system.setting.userDefaultPassword')" path="userDefaultPassword">
      <NInput
        v-model:value="config.userDefaultPassword"
        type="password"
        show-password-on="click"
        :placeholder="$t('page.system.setting.userDefaultPassword')"
        class="max-w-400px"
      />
    </NFormItem>
    <NFormItem :label="$t('page.system.setting.userDefaultRole')" path="userDefaultRole">
      <NInput v-model:value="config.userDefaultRole" :placeholder="$t('page.system.setting.userDefaultRole')" class="max-w-400px" />
    </NFormItem>

    <NDivider />

    <div class="mb-12px text-15px font-500">{{ $t('page.system.setting.captchaTitle') }}</div>
    <NFormItem :label="$t('page.system.setting.enableVerifyCode')" path="enableVerifyCode">
      <NSwitch v-model:value="config.enableVerifyCode" />
    </NFormItem>
    <template v-if="config.enableVerifyCode">
      <NFormItem :label="$t('page.system.setting.verifyCodeType')" path="verifyCodeType">
        <NSelect v-model:value="config.verifyCodeType" :options="captchaTypeOptions" class="max-w-200px" />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.verifyCodeLen')" path="verifyCodeLen">
        <NInputNumber v-model:value="config.verifyCodeLen" :min="2" :max="8" class="max-w-200px">
          <template #suffix>{{ $t('page.system.setting.unitChar') }}</template>
        </NInputNumber>
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.verifyCodeExp')" path="verifyCodeExp">
        <NInputNumber v-model:value="config.verifyCodeExp" :min="1" :max="60" class="max-w-200px">
          <template #suffix>{{ $t('page.system.setting.unitMinute') }}</template>
        </NInputNumber>
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.verifyCodeTokenExp')" path="verifyCodeTokenExp">
        <NInputNumber v-model:value="config.verifyCodeTokenExp" :min="1" :max="60" class="max-w-200px">
          <template #suffix>{{ $t('page.system.setting.unitMinute') }}</template>
        </NInputNumber>
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.verifyInaccuracy')" path="verifyInaccuracy">
        <NInputNumber v-model:value="config.verifyInaccuracy" :min="0" :max="50" class="max-w-200px">
          <template #suffix>{{ $t('page.system.setting.unitPixel') }}</template>
        </NInputNumber>
      </NFormItem>
    </template>

    <NDivider />

    <div class="mb-12px text-15px font-500">{{ $t('page.system.setting.logTitle') }}</div>
    <NFormItem :label="$t('page.system.setting.loginLogRetentionDays')" path="loginLogRetentionDays">
      <NInputNumber v-model:value="config.loginLogRetentionDays" :min="7" :max="365" class="max-w-200px">
        <template #suffix>{{ $t('page.system.setting.unitDay') }}</template>
      </NInputNumber>
    </NFormItem>
    <NFormItem :label="$t('page.system.setting.operationLogRetentionDays')" path="operationLogRetentionDays">
      <NInputNumber v-model:value="config.operationLogRetentionDays" :min="7" :max="365" class="max-w-200px">
        <template #suffix>{{ $t('page.system.setting.unitDay') }}</template>
      </NInputNumber>
    </NFormItem>
  </NForm>
</template>
