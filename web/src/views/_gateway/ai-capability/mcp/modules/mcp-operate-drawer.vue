<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { fetchCreateMCPServer, fetchUpdateMCPServer } from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { MCP_AUTH_TYPE_OPTIONS, MCP_BILLING_OPTIONS, MCP_TRANSPORT_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'MCPOperateDrawer' });

interface Props {
  operateType: NaiveUI.TableOperateType;
  rowData?: Api.Gateway.MCPServer | null;
}

const props = defineProps<Props>();

interface Emits {
  (e: 'submitted'): void;
}

const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', {
  default: false
});

const { formRef, validate, restoreValidation } = useNaiveForm();
const { createRequiredRule } = useFormRules();

const title = computed(() => (props.operateType === 'add' ? $t('page.gateway.mcp.add') : $t('page.gateway.mcp.edit')));

type Model = Api.Gateway.MCPServerOperateParams;

const model = ref<Model>(createDefaultModel());

function createDefaultModel(): Model {
  return {
    mcpServerId: null,
    name: '',
    serverName: '',
    url: '',
    transport: 'streamable_http',
    authType: 'none',
    credentials: {},
    description: '',
    instructions: '',
    category: 'general',
    author: '',
    iconUrl: '',
    documentationUrl: '',
    billingType: 'free',
    externalCostPerCall: null,
    isActive: true
  };
}

const transportOptions = MCP_TRANSPORT_OPTIONS.map(o => ({ label: $t(o.label), value: o.value }));
const authTypeOptions = MCP_AUTH_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value }));
const billingOptions = MCP_BILLING_OPTIONS.map(o => ({ label: $t(o.label), value: o.value }));

/** 是否需要凭据(api_key/bearer_token)；none 无凭据 */
const needsCredentials = computed(() => model.value.authType && model.value.authType !== 'none');

const rules: Record<string, App.Global.FormRule> = {
  name: createRequiredRule($t('page.gateway.mcp.form.name.required')),
  serverName: {
    trigger: ['blur', 'change'],
    validator: (_rule: unknown, value: string) => {
      if (!value) return new Error($t('page.gateway.mcp.form.serverName.required'));
      if (!/^[a-zA-Z0-9_]{1,128}$/.test(value)) {
        return new Error($t('page.gateway.mcp.form.serverName.pattern'));
      }
      return true;
    }
  },
  url: createRequiredRule($t('page.gateway.mcp.form.url.required'))
};

/** 编辑态:凭据掩码回传=保留旧明文;填新值覆盖;清空切换 none 时置 null 清残留 */
const authValue = ref('');

function initModel() {
  if (props.rowData) {
    const row = props.rowData;
    model.value = {
      mcpServerId: row.mcpServerId,
      name: row.name,
      serverName: row.serverName,
      url: row.url,
      transport: row.transport,
      authType: row.authType,
      credentials: row.credentials ? { ...row.credentials } : {},
      description: row.description,
      instructions: row.instructions,
      category: row.category,
      author: row.author,
      iconUrl: row.iconUrl,
      documentationUrl: row.documentationUrl,
      billingType: row.billingType,
      externalCostPerCall: row.externalCostPerCall,
      isActive: row.isActive
    };
    authValue.value = row.credentials?.auth_value ?? '';
  } else {
    model.value = createDefaultModel();
    authValue.value = '';
  }
}

watch(visible, val => {
  if (val) {
    initModel();
    restoreValidation();
  }
});

/** 提交前组装凭据:none 显式清空;其余保留掩码回传的旧值并合并新输入的 auth_value */
function buildCredentials(): Record<string, string> | null {
  if (!needsCredentials.value) return null;
  const credentials: Record<string, string> = { ...model.value.credentials };
  if (authValue.value && !authValue.value.includes('****')) {
    credentials.auth_value = authValue.value;
  }
  return credentials;
}

async function handleSubmit() {
  await validate();
  const payload: Model = {
    ...model.value,
    credentials: buildCredentials()
  };
  const isAdd = props.operateType === 'add';
  const { error } = isAdd ? await fetchCreateMCPServer(payload) : await fetchUpdateMCPServer(payload);
  if (error) return;
  window.$message?.success($t(isAdd ? 'common.addSuccess' : 'common.updateSuccess'));
  emit('submitted');
}
</script>

<template>
  <NDrawer v-model:show="visible" display-directive="show" :width="560" class="max-w-90%">
    <NDrawerContent :title="title" :native-scrollbar="false" closable>
      <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="110">
        <NFormItem :label="$t('page.gateway.mcp.col.name')" path="name">
          <NInput v-model:value="model.name" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.serverName')" path="serverName">
          <NInput
            v-model:value="model.serverName"
            :disabled="operateType === 'edit'"
            :placeholder="$t('common.placeholderInput')"
          />
        </NFormItem>
        <p v-if="operateType === 'add'" class="mb-12px ml-110px text-12px text-slate-400">
          {{ $t('page.gateway.mcp.form.serverName.tip') }}
        </p>
        <p v-else class="mb-12px ml-110px text-12px text-slate-400">
          {{ $t('page.gateway.mcp.form.serverName.renameTip') }}
        </p>
        <NFormItem :label="$t('page.gateway.mcp.col.url')" path="url">
          <NInput v-model:value="model.url" placeholder="https://host/mcp" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.transport')">
          <NRadioGroup v-model:value="model.transport">
            <NRadio v-for="opt in transportOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadio>
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.authType')">
          <NSelect v-model:value="model.authType" :options="authTypeOptions" />
        </NFormItem>
        <template v-if="needsCredentials">
          <NFormItem :label="$t('page.gateway.mcp.form.authValue')">
            <NInput
              v-model:value="authValue"
              type="password"
              show-password-on="click"
              :placeholder="$t('page.gateway.mcp.form.authValuePlaceholder')"
            />
          </NFormItem>
          <p class="mb-12px ml-110px text-12px text-slate-400">{{ $t('page.gateway.mcp.form.valuesTip') }}</p>
        </template>
        <NFormItem :label="$t('page.gateway.mcp.col.billingType')">
          <NRadioGroup v-model:value="model.billingType">
            <NRadio v-for="opt in billingOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadio>
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="model.billingType === 'per_call'" :label="$t('page.gateway.mcp.costPerCall')">
          <NInputNumber v-model:value="model.externalCostPerCall" :min="0" :precision="6" class="w-full" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.category')">
          <NInput v-model:value="model.category" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.author')">
          <NInput v-model:value="model.author" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.iconUrl')">
          <NInput v-model:value="model.iconUrl" placeholder="https://..." />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.documentationUrl')">
          <NInput v-model:value="model.documentationUrl" placeholder="https://..." />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.description')">
          <NInput v-model:value="model.description" type="textarea" :rows="2" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.form.instructions')">
          <NInput v-model:value="model.instructions" type="textarea" :rows="3" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.isActive')">
          <NSwitch :value="!!model.isActive" @update:value="(val: boolean) => (model.isActive = val)" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end" :size="12">
          <NButton @click="visible = false">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped></style>
