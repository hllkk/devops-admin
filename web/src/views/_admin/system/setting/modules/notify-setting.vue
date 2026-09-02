<script setup lang="tsx">
import { computed, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DataTableColumn } from 'naive-ui';
import { NTime } from 'naive-ui';
import UserSelect from '@/components/custom/user-select.vue';
import ButtonIcon from '@/components/custom/button-icon.vue';
import { fetchGetDeptTree, fetchGetUserSelect } from '@/service/api/system';
import {
  fetchTestEmail,
  fetchTestWecomApp,
  fetchTestWecomBot,
  fetchWecomBotGroupCreate,
  fetchWecomBotGroupDelete,
  fetchWecomBotGroupList
} from '@/service/api/system/setting';

defineOptions({ name: 'NotifySetting' });

const { t } = useI18n();

const config = defineModel<Api.System.NotifySettingConfig>('config', { required: true });
const policy = defineModel<Api.System.NotifyPolicyConfig>('policy', { required: true });

const sslModeOptions = [
  { label: t('page.system.setting.notifyEmailSSLModeNone'), value: 'none' },
  { label: t('page.system.setting.notifyEmailSSLModeSSL'), value: 'ssl' },
  { label: t('page.system.setting.notifyEmailSSLModeStartTLS'), value: 'starttls' }
];

// 晨报场景参数兜底(后端 params 可能为空)
if (!policy.value.params) {
  policy.value.params = {};
}
const morningParams = computed(() => policy.value.params ?? {});

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

// --- 群机器人群表格(新增/删除即时生效,不走聚合保存) ---

const botGroups = ref<Api.System.WecomBotGroup[]>([]);

async function loadBotGroups() {
  const { data, error } = await fetchWecomBotGroupList();
  if (!error && data) {
    botGroups.value = data;
  }
}

const groupOptions = computed(() =>
  botGroups.value.map(g => ({ label: g.groupName, value: g.id as CommonType.IdType }))
);

const groupColumns = computed<DataTableColumn<Api.System.WecomBotGroup>[]>(() => [
  {
    title: t('page.system.setting.notifyWecomBotGroupName'),
    key: 'groupName',
    minWidth: 140,
    ellipsis: { tooltip: true }
  },
  {
    title: t('page.system.setting.notifyWecomBotWebhook'),
    key: 'webhookUrl',
    minWidth: 200,
    ellipsis: { tooltip: true }
  },
  {
    title: t('page.system.setting.notifyWecomBotGroupCreatedAt'),
    key: 'createTime',
    width: 170,
    render: row => <NTime time={Date.parse(row.createTime)} format="yyyy-MM-dd HH:mm:ss" />
  },
  {
    title: t('common.operate'),
    key: 'operate',
    width: 80,
    align: 'center',
    render: row => (
      <ButtonIcon
        text
        type="error"
        icon="material-symbols:delete-outline"
        tooltipContent={t('common.delete')}
        popconfirmContent={t('page.system.setting.notifyWecomBotGroupDeleteConfirm')}
        onPositiveClick={() => handleDeleteGroup(row.id)}
      />
    )
  }
]);

// 新增群弹窗
const showGroupModal = ref(false);
const groupForm = reactive({ groupName: '', webhookUrl: '' });
const groupSubmitting = ref(false);

function openGroupModal() {
  groupForm.groupName = '';
  groupForm.webhookUrl = '';
  showGroupModal.value = true;
}

async function handleCreateGroup() {
  if (!groupForm.groupName.trim() || !groupForm.webhookUrl.trim()) {
    window.$message?.warning(t('common.pleaseCheckValue'));
    return;
  }
  groupSubmitting.value = true;
  const { error } = await fetchWecomBotGroupCreate({
    groupName: groupForm.groupName.trim(),
    webhookUrl: groupForm.webhookUrl.trim()
  });
  groupSubmitting.value = false;
  if (error) {
    return; // 具体错误(名称为空/webhook非法)已由请求层弹出
  }
  window.$message?.success(t('common.addSuccess'));
  showGroupModal.value = false;
  await loadBotGroups();
}

async function handleDeleteGroup(id: CommonType.IdType) {
  const { error } = await fetchWecomBotGroupDelete(id);
  if (error) {
    return; // 具体错误已由请求层弹出
  }
  window.$message?.success(t('common.deleteSuccess'));
  await loadBotGroups();
}

// 群机器人测试(下拉选已录入群)
const botTestGroupId = ref<CommonType.IdType | null>(null);
const botTestLoading = ref(false);

async function handleTestWecomBot() {
  if (!botTestGroupId.value) {
    window.$message?.warning(t('page.system.setting.notifyWecomBotTestSelect'));
    return;
  }
  botTestLoading.value = true;
  const { error } = await fetchTestWecomBot({ groupId: botTestGroupId.value });
  if (error) {
    window.$message?.error(t('page.system.setting.notifyWecomBotTestFail'));
  } else {
    window.$message?.success(t('page.system.setting.notifyWecomBotTestSuccess'));
  }
  botTestLoading.value = false;
}

// 晨报目标:部门树(含子部门多选) / 用户多选
const deptOptions = ref<Api.Common.CommonTreeRecord>([]);
const userOptions = ref<CommonType.Option<CommonType.IdType>[]>([]);

const targetTypeOptions = [
  { label: t('page.system.setting.notifyMorningTargetAll'), value: 'all' },
  { label: t('page.system.setting.notifyMorningTargetDepts'), value: 'depts' },
  { label: t('page.system.setting.notifyMorningTargetUsers'), value: 'users' }
];

// 晨报模板变量说明(变量名为代码不需翻译,描述走 i18n)
const templateVars = computed(() => [
  { name: '{{.ProviderName}}', desc: t('page.system.setting.notifyMorningTemplateVarProviderName') },
  { name: '{{.UsedPercent}}', desc: t('page.system.setting.notifyMorningTemplateVarUsedPercent') },
  { name: '{{.Surplus}}', desc: t('page.system.setting.notifyMorningTemplateVarSurplus') },
  { name: '{{.Total}}', desc: t('page.system.setting.notifyMorningTemplateVarTotal') },
  { name: '{{.ResetLine}}', desc: t('page.system.setting.notifyMorningTemplateVarResetLine') },
  { name: '{{.Overdrawn}}', desc: t('page.system.setting.notifyMorningTemplateVarOverdrawn') }
]);

function resetTemplate(field: 'contentTemplate' | 'markdownTemplate') {
  morningParams.value[field] = '';
}

onMounted(async () => {
  const [deptRes, userRes] = await Promise.all([fetchGetDeptTree(), fetchGetUserSelect(), loadBotGroups()]);
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

    <!-- 企业微信(应用消息 + 群机器人) -->
    <NTabPane name="wecom" :tab="t('page.system.setting.tabNotifyWecom')">
      <NForm :model="config" label-placement="left" :label-width="160" class="mt-16px">
        <!-- 应用消息区块 -->
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

        <NDivider />

        <!-- 群机器人区块 -->
        <NFormItem :label="$t('page.system.setting.notifyWecomBotEnabled')" path="wecomBotEnabled">
          <NSwitch v-model:value="config.wecomBotEnabled" />
        </NFormItem>
        <template v-if="config.wecomBotEnabled">
          <NAlert type="info" :show-icon="false" class="mb-12px max-w-560px">
            {{ $t('page.system.setting.notifyWecomBotGroupTip') }}
          </NAlert>
          <div class="flex justify-end mb-8px">
            <NButton type="primary" @click="openGroupModal">
              {{ $t('page.system.setting.notifyWecomBotGroupAdd') }}
            </NButton>
          </div>
          <NDataTable
            :columns="groupColumns"
            :data="botGroups"
            :bordered="false"
            size="small"
            class="max-w-760px mb-16px"
          />
          <NFormItem :label="$t('page.system.setting.notifyWecomBotTest')" path="testBot">
            <div class="flex items-center gap-12px w-full max-w-520px">
              <NSelect
                v-model:value="botTestGroupId"
                clearable
                filterable
                :options="groupOptions"
                :placeholder="$t('page.system.setting.notifyWecomBotTestSelect')"
                class="flex-1"
              />
              <NButton type="primary" :loading="botTestLoading" @click="handleTestWecomBot">
                {{ $t('page.system.setting.notifyWecomBotTestBtn') }}
              </NButton>
            </div>
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
          <!-- 推送渠道:勾哪个发哪个,站内始终发送 -->
          <NFormItem :label="$t('page.system.setting.notifyMorningChannels')" path="channels">
            <div class="flex flex-col gap-4px">
              <NSpace>
                <NCheckbox v-model:checked="morningParams.wecomApp">
                  {{ $t('page.system.setting.notifyMorningChannelWecomApp') }}
                </NCheckbox>
                <NCheckbox v-model:checked="morningParams.wecomBot">
                  {{ $t('page.system.setting.notifyMorningChannelWecomBot') }}
                </NCheckbox>
              </NSpace>
              <span class="text-12px text-gray-400">
                {{ $t('page.system.setting.notifyMorningChannelTip') }}
              </span>
            </div>
          </NFormItem>
          <NFormItem
            v-if="morningParams.wecomBot"
            :label="$t('page.system.setting.notifyMorningBotGroups')"
            path="botGroupIds"
          >
            <NSelect
              v-model:value="morningParams.botGroupIds"
              multiple
              clearable
              filterable
              :options="groupOptions"
              :placeholder="$t('common.placeholderSelect')"
              class="max-w-520px"
            />
          </NFormItem>

          <!-- 正文模板 -->
          <NDivider>{{ $t('page.system.setting.notifyMorningTemplateTitle') }}</NDivider>
          <NFormItem :label="$t('page.system.setting.notifyMorningTemplateVars')" path="templateVars">
            <div class="flex flex-wrap gap-8px max-w-560px">
              <NTag v-for="v in templateVars" :key="v.name" size="small" :bordered="false">
                <span class="font-mono">{{ v.name }}</span>
                <span class="ml-4px text-gray-400">{{ v.desc }}</span>
              </NTag>
            </div>
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyMorningTemplateContent')" path="contentTemplate">
            <div class="w-full max-w-560px">
              <NInput
                v-model:value="morningParams.contentTemplate"
                type="textarea"
                :autosize="{ minRows: 4, maxRows: 10 }"
                :placeholder="$t('page.system.setting.notifyMorningTemplateTip')"
              />
              <div class="flex justify-end mt-4px">
                <NButton size="tiny" quaternary type="primary" @click="resetTemplate('contentTemplate')">
                  {{ $t('page.system.setting.notifyMorningTemplateReset') }}
                </NButton>
              </div>
            </div>
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.notifyMorningTemplateMarkdown')" path="markdownTemplate">
            <div class="w-full max-w-560px">
              <NInput
                v-model:value="morningParams.markdownTemplate"
                type="textarea"
                :autosize="{ minRows: 4, maxRows: 10 }"
                :placeholder="$t('page.system.setting.notifyMorningTemplateTip')"
              />
              <div class="flex justify-end mt-4px">
                <NButton size="tiny" quaternary type="primary" @click="resetTemplate('markdownTemplate')">
                  {{ $t('page.system.setting.notifyMorningTemplateReset') }}
                </NButton>
              </div>
            </div>
          </NFormItem>

          <NAlert type="info" :show-icon="false" class="max-w-560px">
            {{ $t('page.system.setting.notifyMorningTip') }}
          </NAlert>
        </template>
      </NForm>
    </NTabPane>
  </NTabs>

  <!-- 新增群弹窗(须置于 NTabs 外:NTabs 只渲染 NTabPane 子节点,内部其余组件会被丢弃) -->
  <NModal
    v-model:show="showGroupModal"
    preset="card"
    :title="$t('page.system.setting.notifyWecomBotGroupAddTitle')"
    class="w-480px"
  >
    <NForm :model="groupForm" label-placement="left" :label-width="100">
      <NFormItem :label="$t('page.system.setting.notifyWecomBotGroupName')" path="groupName">
        <NInput
          v-model:value="groupForm.groupName"
          :placeholder="$t('page.system.setting.notifyWecomBotGroupNamePlaceholder')"
        />
      </NFormItem>
      <NFormItem :label="$t('page.system.setting.notifyWecomBotWebhook')" path="webhookUrl">
        <NInput
          v-model:value="groupForm.webhookUrl"
          :placeholder="$t('page.system.setting.notifyWecomBotWebhookPlaceholder')"
        />
      </NFormItem>
      <div class="flex justify-end gap-12px">
        <NButton @click="showGroupModal = false">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="groupSubmitting" @click="handleCreateGroup">
          {{ $t('common.confirm') }}
        </NButton>
      </div>
    </NForm>
  </NModal>
</template>
