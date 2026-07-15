<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

defineOptions({ name: 'SecuritySetting' });

const { t } = useI18n();

const config = defineModel<Api.System.SecuritySettingConfig>('config', { required: true });

const ipModeOptions = computed(() => [
  { label: t('page.system.setting.ipMode.blacklist'), value: 'blacklist' },
  { label: t('page.system.setting.ipMode.whitelist'), value: 'whitelist' }
]);

const isBlacklistMode = computed(() => config.value.ipValidationMode === 'blacklist');
const isWhitelistMode = computed(() => config.value.ipValidationMode === 'whitelist');
</script>

<template>
  <div>
    <div class="mb-12px text-15px font-500">{{ $t('page.system.setting.passwordTitle') }}</div>
    <NForm :model="config" label-placement="left" :label-width="160">
      <NFormItem :label="$t('page.system.setting.passwordMinLength')" path="passwordMinLength">
        <NInputNumber v-model:value="config.passwordMinLength" :min="6" :max="32" class="max-w-200px" />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.passwordRequireUppercase')" path="passwordRequireUppercase">
        <NSwitch v-model:value="config.passwordRequireUppercase" />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.passwordRequireLowercase')" path="passwordRequireLowercase">
        <NSwitch v-model:value="config.passwordRequireLowercase" />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.passwordRequireDigit')" path="passwordRequireDigit">
        <NSwitch v-model:value="config.passwordRequireDigit" />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.passwordRequireSpecial')" path="passwordRequireSpecial">
        <NSwitch v-model:value="config.passwordRequireSpecial" />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.loginFailLockCount')" path="loginFailLockCount">
        <NInputNumber v-model:value="config.loginFailLockCount" :min="1" :max="10" class="max-w-200px" />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.loginFailLockTime')" path="loginFailLockTime">
        <NInputNumber v-model:value="config.loginFailLockTime" :min="1" :max="1440" class="max-w-200px">
          <template #suffix>{{ $t('page.system.setting.unitMinute') }}</template>
        </NInputNumber>
      </NFormItem>
    </NForm>

    <NDivider />

    <div class="mb-12px text-15px font-500">{{ $t('page.system.setting.ipTitle') }}</div>
    <NForm :model="config" label-placement="left" :label-width="160">
      <NFormItem :label="$t('page.system.setting.ipValidationEnabled')" path="ipValidationEnabled">
        <NSwitch v-model:value="config.ipValidationEnabled" />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.ipValidationMode')" path="ipValidationMode">
        <NSelect v-model:value="config.ipValidationMode" :options="ipModeOptions" class="max-w-200px" />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.ipBlacklist')" path="ipBlacklist">
        <NInput
          v-model:value="config.ipBlacklist"
          type="textarea"
          :placeholder="$t('page.system.setting.ipListPlaceholder')"
          :rows="4"
          :disabled="isWhitelistMode"
          class="max-w-400px"
        />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.ipWhitelist')" path="ipWhitelist">
        <NInput
          v-model:value="config.ipWhitelist"
          type="textarea"
          :placeholder="$t('page.system.setting.ipListPlaceholder')"
          :rows="4"
          :disabled="isBlacklistMode"
          class="max-w-400px"
        />
      </NFormItem>
    </NForm>
  </div>
</template>
