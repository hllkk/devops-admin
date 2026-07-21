<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { fetchTestEmail } from '@/service/api/system/setting';

defineOptions({ name: 'NotifySetting' });

const { t } = useI18n();

const config = defineModel<Api.System.NotifySettingConfig>('config', { required: true });

const sslModeOptions = [
  { label: t('page.system.setting.notifyEmailSSLModeNone'), value: 'none' },
  { label: t('page.system.setting.notifyEmailSSLModeSSL'), value: 'ssl' },
  { label: t('page.system.setting.notifyEmailSSLModeStartTLS'), value: 'starttls' }
];

// 测试邮件
const testTo = ref('');
const testLoading = ref(false);

async function handleTestEmail() {
  if (!testTo.value) {
    window.$message?.warning(t('page.system.setting.notifyTestEmailToPlaceholder'));
    return;
  }
  testLoading.value = true;
  const { error } = await fetchTestEmail({
    emailHost: config.value.emailHost,
    emailPort: config.value.emailPort,
    emailUsername: config.value.emailUsername,
    emailPassword: config.value.emailPassword,
    emailFromAddr: config.value.emailFromAddr,
    emailFromName: config.value.emailFromName,
    emailSSLMode: config.value.emailSSLMode,
    testTo: testTo.value
  });
  if (error) {
    window.$message?.error(t('page.system.setting.notifyTestEmailFail'));
  } else {
    window.$message?.success(t('page.system.setting.notifyTestEmailSuccess'));
  }
  testLoading.value = false;
}
</script>

<template>
  <NTabs type="line" animated>
    <!-- 邮件通知 -->
    <NTabPane name="email" :tab="t('page.system.setting.tabNotifyEmail')">
      <NForm :model="config" label-placement="left" :label-width="160" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.notifyEmailEnabled')" path="emailEnabled">
          <NSwitch v-model:value="config.emailEnabled" />
        </NFormItem>
        <template v-if="config.emailEnabled">
          <NFormItem :label="$t('page.system.setting.notifyEmailHost')" path="emailHost">
            <NInput
              v-model:value="config.emailHost"
              :placeholder="$t('page.system.setting.notifyEmailHostPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyEmailPort')" path="emailPort">
            <NInputNumber v-model:value="config.emailPort" :min="1" :max="65535" class="max-w-200px" />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyEmailUsername')" path="emailUsername">
            <NInput
              v-model:value="config.emailUsername"
              :placeholder="$t('page.system.setting.notifyEmailUsernamePlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyEmailPassword')" path="emailPassword">
            <NInput
              v-model:value="config.emailPassword"
              type="password"
              show-password-on="click"
              :placeholder="$t('page.system.setting.notifyEmailPasswordPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyEmailFromAddr')" path="emailFromAddr">
            <NInput
              v-model:value="config.emailFromAddr"
              :placeholder="$t('page.system.setting.notifyEmailFromAddrPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyEmailFromName')" path="emailFromName">
            <NInput
              v-model:value="config.emailFromName"
              :placeholder="$t('page.system.setting.notifyEmailFromNamePlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyEmailSSLMode')" path="emailSSLMode">
            <NSelect
              v-model:value="config.emailSSLMode"
              :options="sslModeOptions"
              class="max-w-200px"
            />
          </NFormItem>
          <NDivider />
          <NFormItem :label="$t('page.system.setting.notifyTestEmailTo')" path="testTo">
            <div class="flex items-center gap-12px w-full max-w-520px">
              <NInput
                v-model:value="testTo"
                :placeholder="$t('page.system.setting.notifyTestEmailToPlaceholder')"
                class="flex-1"
              />
              <NButton type="primary" :loading="testLoading" @click="handleTestEmail">
                {{ testLoading ? $t('page.system.setting.notifyTestEmailSending') : $t('page.system.setting.notifyTestEmail') }}
              </NButton>
            </div>
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>

    <!-- Webhook -->
    <NTabPane name="webhook" :tab="t('page.system.setting.tabNotifyWebhook')">
      <NForm :model="config" label-placement="left" :label-width="160" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.notifyWebhookEnabled')" path="webhookEnabled">
          <NSwitch v-model:value="config.webhookEnabled" />
        </NFormItem>
        <template v-if="config.webhookEnabled">
          <NFormItem :label="$t('page.system.setting.notifyWebhookUrl')" path="webhookUrl">
            <NInput
              v-model:value="config.webhookUrl"
              :placeholder="$t('page.system.setting.notifyWebhookUrlPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyWebhookSecret')" path="webhookSecret">
            <NInput
              v-model:value="config.webhookSecret"
              type="password"
              show-password-on="click"
              :placeholder="$t('page.system.setting.notifyWebhookSecretPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>
  </NTabs>
</template>
