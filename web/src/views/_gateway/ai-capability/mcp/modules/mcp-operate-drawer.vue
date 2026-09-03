<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { SelectOption } from 'naive-ui';
import { fetchCreateMCPServer, fetchGetMCPCategories, fetchUpdateMCPServer } from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { MCP_AUTH_TYPE_OPTIONS, MCP_BILLING_OPTIONS, MCP_STDIO_COMMAND_OPTIONS, MCP_TRANSPORT_OPTIONS } from '@/constants/business/gateway';

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
    command: '',
    args: [],
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
    internalCostPerCall: null,
    isActive: true
  };
}

const transportOptions = MCP_TRANSPORT_OPTIONS.map(o => ({ label: $t(o.label), value: o.value }));
const commandOptions = MCP_STDIO_COMMAND_OPTIONS.map(c => ({ label: c, value: c }));
const authTypeOptions = MCP_AUTH_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value }));
const billingOptions = MCP_BILLING_OPTIONS.map(o => ({ label: $t(o.label), value: o.value }));

/** stdio 型：本地子进程由 LiteLLM 容器托管，凭据走 env 变量 */
const isStdio = computed(() => model.value.transport === 'stdio');

/** 是否需要凭据(api_key/bearer_token)；none 无凭据；stdio 恒 none */
const needsCredentials = computed(() => !isStdio.value && model.value.authType && model.value.authType !== 'none');

// computed：url/command 的必填星号随 transport 形态切换显隐(校验逻辑本身在 validator 内响应)
const rules = computed<Record<string, App.Global.FormRule>>(() => ({
  name: createRequiredRule($t('page.gateway.mcp.form.name.required')),
  serverName: {
    required: true,
    trigger: ['blur', 'change'],
    validator: (_rule: unknown, value: string) => {
      if (!value) return new Error($t('page.gateway.mcp.form.serverName.required'));
      if (!/^[a-zA-Z0-9_]{1,128}$/.test(value)) {
        return new Error($t('page.gateway.mcp.form.serverName.pattern'));
      }
      return true;
    }
  },
  url: {
    required: !isStdio.value,
    trigger: ['blur', 'change'],
    validator: (_rule: unknown, value: string) => {
      if (isStdio.value) return true;
      if (!value) return new Error($t('page.gateway.mcp.form.url.required'));
      return true;
    }
  },
  command: {
    required: isStdio.value,
    trigger: ['blur', 'change'],
    validator: (_rule: unknown, value: string) => {
      if (!isStdio.value) return true;
      if (!value) return new Error($t('page.gateway.mcp.form.command.required'));
      return true;
    }
  }
}));

/** stdio env 编辑行(从掩码凭据视图初始化；掩码原样回传=后端保留旧明文) */
interface EnvRow {
  key: string;
  value: string;
}

const envRows = ref<EnvRow[]>([]);

/** NDynamicInput 需可变数组(null 容忍双向收窄) */
const argsInput = computed<string[]>({
  get: () => model.value.args ?? [],
  set: val => {
    model.value.args = val;
  }
});

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
      command: row.command ?? '',
      args: row.args ?? [],
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
      internalCostPerCall: row.internalCostPerCall,
      isActive: row.isActive
    };
    authValue.value = row.credentials?.auth_value ?? '';
    // stdio:env 行从掩码凭据视图初始化(掩码回传=保留旧明文)
    envRows.value = Object.entries(row.credentials ?? {}).map(([key, value]) => ({ key, value: String(value) }));
  } else {
    model.value = createDefaultModel();
    authValue.value = '';
    envRows.value = [];
  }
}

watch(visible, val => {
  if (val) {
    initModel();
    restoreValidation();
    loadCategories();
  }
});

// ── 分类受控下拉(现有值可选,新值经 tag 输入;与 Skill 抽屉同口径) ──

const categoryOptions = ref<SelectOption[]>([]);

/** NSelect 值(空串归一 null 以显示 placeholder；新值经 tag 输入回写 model) */
const categoryValue = computed<string | null>({
  get: () => model.value.category || null,
  set: val => {
    model.value.category = val ?? '';
  }
});

/** 拉取现有分类(distinct)做下拉选项;失败静默(不影响表单可用性,退化为手输) */
async function loadCategories() {
  const { error, data } = await fetchGetMCPCategories();
  if (!error && data) {
    categoryOptions.value = data.map(c => ({ label: c, value: c }));
  }
}

/** 提交前组装凭据:http 型 none 显式清空,其余保留掩码回传旧值并合并新输入 auth_value；
 *  stdio 型组装 env 键值对(空=未提交保留旧值,掩码原样回传=保留旧明文) */
function buildCredentials(): Record<string, string> | null {
  if (isStdio.value) {
    const credentials: Record<string, string> = {};
    for (const item of envRows.value) {
      const key = item.key.trim();
      if (key) credentials[key] = item.value;
    }
    return Object.keys(credentials).length ? credentials : null;
  }
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
  const { error, data } = isAdd ? await fetchCreateMCPServer(payload) : await fetchUpdateMCPServer(payload);
  if (error) return;
  // 创建即自动拉取工具(后端串联)：按返回工具数给出反馈，未拉到引导去工具面板重试
  if (isAdd) {
    if (data?.toolCount) {
      window.$message?.success($t('page.gateway.mcp.toolsDrawer.refreshSuccess', { count: data.toolCount }));
    } else {
      window.$message?.warning($t('page.gateway.mcp.form.addToolsMissed'));
    }
  } else {
    window.$message?.success($t('common.updateSuccess'));
  }
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
        <NFormItem :label="$t('page.gateway.mcp.col.transport')">
          <NRadioGroup v-model:value="model.transport">
            <NRadio v-for="opt in transportOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadio>
          </NRadioGroup>
        </NFormItem>
        <template v-if="!isStdio">
          <NFormItem :label="$t('page.gateway.mcp.col.url')" path="url">
            <NInput v-model:value="model.url" placeholder="https://host/mcp" />
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
        </template>
        <template v-else>
          <NFormItem :label="$t('page.gateway.mcp.col.command')" path="command">
            <NSelect v-model:value="model.command" :options="commandOptions" :placeholder="$t('common.placeholderSelect')" />
          </NFormItem>
          <p class="mb-12px ml-110px text-12px text-slate-400">{{ $t('page.gateway.mcp.form.commandTip') }}</p>
          <NFormItem :label="$t('page.gateway.mcp.col.args')">
            <NDynamicInput v-model:value="argsInput" :placeholder="$t('page.gateway.mcp.form.argsPlaceholder')" />
          </NFormItem>
          <NFormItem :label="$t('page.gateway.mcp.col.env')">
            <div class="w-full">
              <NDynamicInput v-model:value="envRows" :on-create="() => ({ key: '', value: '' })">
                <template #default="{ value }">
                  <div class="flex w-full items-center gap-8px">
                    <NInput v-model:value="value.key" :placeholder="$t('page.gateway.mcp.form.envKeyPlaceholder')" />
                    <NInput v-model:value="value.value" :placeholder="$t('page.gateway.mcp.form.envValuePlaceholder')" />
                  </div>
                </template>
              </NDynamicInput>
              <p class="mt-4px text-12px text-slate-400">{{ $t('page.gateway.mcp.form.envTip') }}</p>
            </div>
          </NFormItem>
        </template>
        <NFormItem :label="$t('page.gateway.mcp.col.billingType')">
          <NRadioGroup v-model:value="model.billingType">
            <NRadio v-for="opt in billingOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadio>
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="model.billingType === 'per_call'" :label="$t('page.gateway.mcp.costPerCall')">
          <NInputNumber v-model:value="model.externalCostPerCall" :min="0" :precision="6" class="w-full" />
        </NFormItem>
        <NFormItem
          v-if="model.billingType === 'per_call'"
          :label="$t('page.gateway.mcp.internalCostPerCall')"
          :feedback="$t('page.gateway.mcp.internalCostPerCallTip')"
        >
          <NInputNumber v-model:value="model.internalCostPerCall" :min="0" :precision="6" class="w-full" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.category')">
          <NSelect
            v-model:value="categoryValue"
            :options="categoryOptions"
            :placeholder="$t('page.gateway.mcp.form.categoryPlaceholder')"
            filterable
            tag
            clearable
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.author')">
          <NInput v-model:value="model.author" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.mcp.col.iconUrl')">
          <IconPicker v-model:value="model.iconUrl" />
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
