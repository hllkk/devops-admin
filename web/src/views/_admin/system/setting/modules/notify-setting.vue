<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import UserSelect from '@/components/custom/user-select.vue';
import { fetchGetDeptTree, fetchGetUserSelect } from '@/service/api/system';
import { fetchTestEmail, fetchTestWecomApp, fetchTestWecomBot } from '@/service/api/system/setting';

defineOptions({ name: 'NotifySetting' });

const { t } = useI18n();

const config = defineModel<Api.System.NotifySettingConfig>('config', { required: true });
const policy = defineModel<Api.System.NotifyPolicyConfig>('policy', { required: true });

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

// 企微应用消息测试(选人实测,凭证取已保存的认证配置)
const wecomTestUserId = ref<CommonType.IdType | null>(null);
const wecomTestLoading = ref(false);

async function handleTestWecomApp() {
  if (!wecomTestUserId.value) {
    window.$message?.warning(t('page.system.setting.notifyWecomTestUserPlaceholder'));
    return;
  }
  wecomTestLoading.value = true;
  const { error } = await fetchTestWecomApp({
    testUserId: wecomTestUserId.value,
    redirectBase: config.value.wecomPushRedirectBase
  });
  if (error) {
    window.$message?.error(t('page.system.setting.notifyWecomTestFail'));
  } else {
    window.$message?.success(t('page.system.setting.notifyWecomTestSuccess'));
  }
  wecomTestLoading.value = false;
}

// 群机器人测试(用当前表单 webhook,未保存也可测)
const wecomBotTestLoading = ref(false);

async function handleTestWecomBot() {
  if (!config.value.wecomBotWebhook) {
    window.$message?.warning(t('page.system.setting.notifyWecomBotWebhookPlaceholder'));
    return;
  }
  wecomBotTestLoading.value = true;
  const { error } = await fetchTestWecomBot({ webhookUrl: config.value.wecomBotWebhook });
  if (error) {
    window.$message?.error(t('page.system.setting.notifyWecomBotTestFail'));
  } else {
    window.$message?.success(t('page.system.setting.notifyWecomBotTestSuccess'));
  }
  wecomBotTestLoading.value = false;
}

// 晨报目标:部门树(含子部门多选) / 用户多选
const deptOptions = ref<Api.Common.CommonTreeRecord>([]);
const userOptions = ref<CommonType.Option<CommonType.IdType>[]>([]);

const targetTypeOptions = [
  { label: t('page.system.setting.notifyMorningTargetAll'), value: 'all' },
  { label: t('page.system.setting.notifyMorningTargetDepts'), value: 'depts' },
  { label: t('page.system.setting.notifyMorningTargetUsers'), value: 'users' }
];

onMounted(async () => {
  const [deptRes, userRes] = await Promise.all([fetchGetDeptTree(), fetchGetUserSelect()]);
  if (!deptRes.error && deptRes.data) {
    deptOptions.value = deptRes.data;
  }
  if (!userRes.error && userRes.data) {
    userOptions.value = userRes.data.map(item => ({
      label: `${item.nickName} ( ${item.userName} )`,
      value: item.userId
    }));
  }
});
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
            <NSelect v-model:value="config.emailSSLMode" :options="sslModeOptions" class="max-w-200px" />
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
                {{
                  testLoading
                    ? $t('page.system.setting.notifyTestEmailSending')
                    : $t('page.system.setting.notifyTestEmail')
                }}
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

    <!-- 企业微信应用消息 -->
    <NTabPane name="wecom" :tab="t('page.system.setting.tabNotifyWecom')">
      <NForm :model="config" label-placement="left" :label-width="160" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.notifyWecomPushEnabled')" path="wecomPushEnabled">
          <NSwitch v-model:value="config.wecomPushEnabled" />
        </NFormItem>
        <template v-if="config.wecomPushEnabled">
          <NFormItem :label="$t('page.system.setting.notifyWecomRedirectBase')" path="wecomPushRedirectBase">
            <NInput
              v-model:value="config.wecomPushRedirectBase"
              :placeholder="$t('page.system.setting.notifyWecomRedirectBasePlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyWecomMaxTargets')" path="wecomPushMaxTargets">
            <NInputNumber v-model:value="config.wecomPushMaxTargets" :min="1" :max="10000" class="max-w-200px" />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyPushBudgetAlertEnabled')" path="pushBudgetAlertEnabled">
            <NSwitch v-model:value="config.pushBudgetAlertEnabled" />
          </NFormItem>
          <NAlert type="info" :show-icon="false" class="mb-12px max-w-560px">
            {{ $t('page.system.setting.notifyWecomPushTip') }}
          </NAlert>
          <NDivider />
          <NFormItem :label="$t('page.system.setting.notifyWecomTestUser')" path="testUserId">
            <div class="flex items-center gap-12px w-full max-w-520px">
              <UserSelect
                v-model:value="wecomTestUserId"
                class="flex-1"
                :placeholder="$t('page.system.setting.notifyWecomTestUserPlaceholder')"
                filterable
              />
              <NButton type="primary" :loading="wecomTestLoading" @click="handleTestWecomApp">
                {{ $t('page.system.setting.notifyWecomTestBtn') }}
              </NButton>
            </div>
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>

    <!-- 企业微信群机器人 -->
    <NTabPane name="wecomBot" :tab="t('page.system.setting.tabNotifyWecomBot')">
      <NForm :model="config" label-placement="left" :label-width="160" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.notifyWecomBotEnabled')" path="wecomBotEnabled">
          <NSwitch v-model:value="config.wecomBotEnabled" />
        </NFormItem>
        <template v-if="config.wecomBotEnabled">
          <NFormItem :label="$t('page.system.setting.notifyWecomBotWebhook')" path="wecomBotWebhook">
            <NInput
              v-model:value="config.wecomBotWebhook"
              :placeholder="$t('page.system.setting.notifyWecomBotWebhookPlaceholder')"
              class="max-w-520px"
            />
          </NFormItem>
          <NDivider />
          <NFormItem :label="$t('page.system.setting.notifyWecomBotTest')" path="testBot">
            <NButton type="primary" :loading="wecomBotTestLoading" @click="handleTestWecomBot">
              {{ $t('page.system.setting.notifyWecomBotTestBtn') }}
            </NButton>
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>

    <!-- TokenPlan 晨报 -->
    <NTabPane name="morning" :tab="t('page.system.setting.tabNotifyMorning')">
      <NForm :model="policy" label-placement="left" :label-width="160" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.notifyMorningEnabled')" path="enabled">
          <NSwitch v-model:value="policy.enabled" />
        </NFormItem>
        <template v-if="policy.enabled">
          <NFormItem :label="$t('page.system.setting.notifyMorningSendTime')" path="sendTime">
            <NTimePicker
              v-model:formatted-value="policy.sendTime"
              format="HH:mm"
              :minute-step="1"
              class="max-w-160px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyMorningTargetType')" path="targetType">
            <NRadioGroup v-model:value="policy.targetType">
              <NRadio v-for="opt in targetTypeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadio>
            </NRadioGroup>
          </NFormItem>
          <NFormItem
            v-if="policy.targetType === 'depts'"
            :label="$t('page.system.setting.notifyMorningTargetIds')"
            path="targetIds"
          >
            <NTreeSelect
              v-model:value="policy.targetIds"
              multiple
              checkable
              cascade
              clearable
              filterable
              key-field="id"
              label-field="label"
              :options="deptOptions as []"
              class="max-w-520px"
            />
          </NFormItem>
          <NFormItem
            v-else-if="policy.targetType === 'users'"
            :label="$t('page.system.setting.notifyMorningTargetIds')"
            path="targetIds"
          >
            <NSelect
              v-model:value="policy.targetIds"
              multiple
              clearable
              filterable
              :options="userOptions"
              :placeholder="$t('common.placeholderSelect')"
              class="max-w-520px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyMorningPushEnabled')" path="pushMorningReportEnabled">
            <NSwitch v-model:value="config.pushMorningReportEnabled" />
          </NFormItem>
          <NAlert type="info" :show-icon="false" class="max-w-560px">
            {{ $t('page.system.setting.notifyMorningTip') }}
          </NAlert>
        </template>
      </NForm>
    </NTabPane>
  </NTabs>
</template>
