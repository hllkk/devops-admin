<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { jsonClone } from '@sa/utils';
import { fetchCreateAiKey, fetchGetAllKeyScenarios, fetchGetAvailableMcps, fetchGetAvailableModels, fetchUpdateAiKey } from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import UserSelect from '@/components/custom/user-select.vue';
import DeptTreeSelect from '@/components/custom/dept-tree-select.vue';
import {
  BUDGET_DURATION_OPTIONS,
  KEY_TYPE_OPTIONS,
  OWNER_TYPE_OPTIONS,
  RATE_LIMIT_MODE_OPTIONS,
  getKeyTypeDescKey,
  isMainKeyType
} from '@/constants/business/gateway';

defineOptions({ name: 'AiKeyOperateDrawer' });

interface Props {
  /** the type of operation */
  operateType: NaiveUI.TableOperateType;
  /** the edit row data */
  rowData?: Api.Gateway.AiKey | null;
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

const title = computed(() => (props.operateType === 'add' ? $t('page.gateway.aiKey.add') : $t('page.gateway.aiKey.edit')));

type Model = Api.Gateway.AiKeyOperateParams;

const model = ref<Model>(createDefaultModel());

function createDefaultModel(): Model {
  return {
    aiKeyId: null,
    keyType: 'personal_scene',
    ownerType: 'user',
    ownerId: null,
    name: '',
    description: '',
    scenarioId: null,
    models: [],
    mcps: [],
    modelBudgets: {},
    budgetLimit: null,
    budgetHardLimit: false,
    budgetDuration: '30d',
    rateLimitMode: 'none',
    tpmLimit: null,
    rpmLimit: null,
    modelLimits: {},
    isActive: true,
    expiresAt: null
  };
}

const modelOptions = ref<{ label: string; value: string }[]>([]);
const mcpOptions = ref<{ label: string; value: string }[]>([]);
const scenarioOptions = ref<{ label: string; value: string }[]>([]);

async function loadModels() {
  const { data, error } = await fetchGetAvailableModels();
  if (!error && data) {
    modelOptions.value = data.map(m => ({ label: m.name, value: m.modelKey }));
  }
}

async function loadMcps() {
  const { data, error } = await fetchGetAvailableMcps();
  if (!error && data) {
    // 授权锚点是 serverName(与 LiteLLM allowed_mcp_servers 对齐)，label 附展示名
    mcpOptions.value = data.map(m => ({ label: `${m.name} (${m.serverName})`, value: m.serverName }));
  }
}

async function loadScenarios() {
  const { data, error } = await fetchGetAllKeyScenarios();
  if (!error && data) {
    // value 用字符串(后端 scenarioId 经 json:",string" 出网/入网，保持传输类型闭环)
    scenarioOptions.value = data.map(s => ({ label: s.name, value: String(s.scenarioId) }));
  }
}

onMounted(loadModels);
onMounted(loadMcps);
onMounted(loadScenarios);

const keyTypeOptions = computed(() => KEY_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const ownerTypeOptions = computed(() => OWNER_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const rateLimitModeOptions = computed(() => RATE_LIMIT_MODE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const budgetDurationOptions = computed(() => BUDGET_DURATION_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

const isMainKey = computed(() => isMainKeyType(model.value.keyType));

/** 选中密钥类型后展示的用途说明(动态 i18n) */
const keyTypeDesc = computed(() => {
  const k = getKeyTypeDescKey(model.value.keyType);
  return k ? $t(k) : '';
});

/** 归属对象选择器 placeholder(随 ownerType 切换) */
const ownerPlaceholder = computed(() =>
  model.value.ownerType === 'dept'
    ? $t('page.gateway.aiKey.form.ownerDeptPlaceholder')
    : $t('page.gateway.aiKey.form.ownerUserPlaceholder')
);

const rules: Record<'keyType' | 'ownerType' | 'ownerId' | 'scenarioId', App.Global.FormRule> = {
  keyType: createRequiredRule($t('page.gateway.aiKey.form.keyType.required')),
  ownerType: createRequiredRule($t('page.gateway.aiKey.form.ownerType.required')),
  ownerId: createRequiredRule($t('page.gateway.aiKey.form.ownerId.required')),
  // 场景 Key 必选场景(主 Key 无场景；存量未挂场景的 Key 编辑时引导补选)
  scenarioId: {
    validator: (_rule, value: string | null) => {
      if (!isMainKey.value && !value) {
        return new Error($t('page.gateway.aiKey.form.scenarioRequired'));
      }
      return true;
    }
  }
};

// 切换归属类型时清空已选归属对象(避免 user 的 id 带到 dept；仅新增态)
watch(
  () => model.value.ownerType,
  () => {
    if (props.operateType === 'add') model.value.ownerId = null;
  }
);

// 主 Key 名称固定为 main(对齐后端 buildKeyAlias)；切回场景 Key 时清掉残留的 main；
// 主 Key 无场景，切换时清掉已选场景
watch(
  () => model.value.keyType,
  kt => {
    if (isMainKeyType(kt)) {
      model.value.name = 'main';
      model.value.scenarioId = null;
    } else if (model.value.name === 'main') {
      model.value.name = '';
    }
  }
);

/** 最近一次由选场景自动带出的名称(用户手改过则不再跟随) */
let lastAutoName = '';

// 选场景时名称默认跟随场景名(可手改)；名称为空或仍是上次带出值时才跟随
function handleScenarioChange(value: string | null) {
  model.value.scenarioId = value;
  const opt = scenarioOptions.value.find(o => o.value === value);
  if (opt && (!model.value.name || model.value.name === lastAutoName)) {
    model.value.name = opt.label;
    lastAutoName = opt.label;
  }
}

function handleUpdateModelWhenEdit() {
  model.value = createDefaultModel();
  lastAutoName = '';

  if (props.operateType === 'edit' && props.rowData) {
    Object.assign(model.value, jsonClone(props.rowData));
    // 后端无场景出网 "0"，转 null(下拉显示原始 0 且绕过必选校验)
    if (model.value.scenarioId === '0' || model.value.scenarioId === 0) {
      model.value.scenarioId = null;
    }
  }
}

function closeDrawer() {
  visible.value = false;
}

async function handleSubmit() {
  await validate();

  if (props.operateType === 'add') {
    const { error } = await fetchCreateAiKey(model.value);
    if (error) return;
    window.$message?.success($t('common.addSuccess'));
  }

  if (props.operateType === 'edit') {
    const { error } = await fetchUpdateAiKey(model.value);
    if (error) return;
    window.$message?.success($t('common.updateSuccess'));
  }

  closeDrawer();
  emit('submitted');
}

watch(visible, () => {
  if (visible.value) {
    handleUpdateModelWhenEdit();
    restoreValidation();
  }
});
</script>

<template>
  <NDrawer v-model:show="visible" :title="title" display-directive="show" :width="640" class="max-w-90%">
    <NDrawerContent :title="title" :native-scrollbar="false" closable>
      <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="120">
        <!-- 基础信息(创建必填且不可改) -->
        <NDivider>{{ $t('page.gateway.aiKey.baseSection') }}</NDivider>
        <NFormItem :label="$t('page.gateway.aiKey.col.keyType')" path="keyType">
          <NSelect v-model:value="model.keyType" :disabled="operateType === 'edit'" :options="keyTypeOptions" :placeholder="$t('common.placeholderSelect')" />
        </NFormItem>
        <div v-if="keyTypeDesc" class="mb-16px ml-120px text-12px text-slate-400">{{ keyTypeDesc }}</div>
        <NFormItem :label="$t('page.gateway.aiKey.col.ownerType')" path="ownerType">
          <NSelect v-model:value="model.ownerType" :disabled="operateType === 'edit'" :options="ownerTypeOptions" :placeholder="$t('common.placeholderSelect')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.ownerId')" path="ownerId">
          <UserSelect
            v-if="model.ownerType === 'user'"
            v-model:value="model.ownerId"
            :disabled="operateType === 'edit'"
            :placeholder="ownerPlaceholder"
            class="w-full"
          />
          <DeptTreeSelect
            v-else-if="model.ownerType === 'dept'"
            v-model:value="model.ownerId"
            :disabled="operateType === 'edit'"
            :placeholder="ownerPlaceholder"
            class="w-full"
          />
          <NInputNumber v-else :disabled="true" :placeholder="$t('page.gateway.aiKey.form.ownerType.required')" class="w-full" />
        </NFormItem>
        <NFormItem v-if="!isMainKey" :label="$t('page.gateway.aiKey.col.scenario')" path="scenarioId">
          <NSelect
            :value="model.scenarioId"
            :options="scenarioOptions"
            filterable
            :placeholder="$t('page.gateway.aiKey.form.scenarioPlaceholder')"
            @update:value="handleScenarioChange"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.name')" path="name">
          <NInput
            v-model:value="model.name"
            :disabled="isMainKey"
            :placeholder="isMainKey ? $t('page.gateway.aiKey.form.mainKeyNameFixed') : $t('page.gateway.aiKey.form.namePlaceholder')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.description')" path="description">
          <NInput v-model:value="model.description" type="textarea" :rows="2" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>

        <!-- 授权 -->
        <NDivider>{{ $t('page.gateway.aiKey.authSection') }}</NDivider>
        <NFormItem :label="$t('page.gateway.aiKey.col.models')" path="models">
          <NSelect
            v-model:value="model.models"
            multiple
            filterable
            :options="modelOptions"
            :placeholder="$t('page.gateway.aiKey.form.modelsPlaceholder')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.mcps')" path="mcps">
          <NSelect
            v-model:value="model.mcps"
            multiple
            filterable
            :options="mcpOptions"
            :placeholder="$t('page.gateway.aiKey.form.mcpsPlaceholder')"
          />
        </NFormItem>

        <!-- 预算 -->
        <NDivider>{{ $t('page.gateway.aiKey.budgetSection') }}</NDivider>
        <NFormItem :label="$t('page.gateway.aiKey.col.budgetLimit')" path="budgetLimit">
          <NInputNumber v-model:value="model.budgetLimit" :min="0" clearable :placeholder="$t('page.gateway.common.unlimited')" class="w-full">
            <template #suffix>¥</template>
          </NInputNumber>
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.budgetHardLimit')" path="budgetHardLimit">
          <NSpace align="center" :size="8" :wrap="false">
            <NSwitch :value="!!model.budgetHardLimit" @update:value="v => (model.budgetHardLimit = v)" />
            <span class="text-12px text-slate-400">{{ $t('page.gateway.aiKey.form.budgetHardLimitDesc') }}</span>
          </NSpace>
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.budgetDuration')" path="budgetDuration">
          <NSelect v-model:value="model.budgetDuration" :options="budgetDurationOptions" :placeholder="$t('common.placeholderSelect')" />
        </NFormItem>

        <!-- 限流 -->
        <NDivider>{{ $t('page.gateway.aiKey.rateLimitSection') }}</NDivider>
        <NFormItem :label="$t('page.gateway.aiKey.col.rateLimitMode')" path="rateLimitMode">
          <NSelect v-model:value="model.rateLimitMode" :options="rateLimitModeOptions" :placeholder="$t('common.placeholderSelect')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.tpmLimit')" path="tpmLimit">
          <NInputNumber v-model:value="model.tpmLimit" :min="0" clearable :placeholder="$t('page.gateway.common.unlimited')" class="w-full" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.rpmLimit')" path="rpmLimit">
          <NInputNumber v-model:value="model.rpmLimit" :min="0" clearable :placeholder="$t('page.gateway.common.unlimited')" class="w-full" />
        </NFormItem>

        <NFormItem :label="$t('page.gateway.aiKey.col.isActive')" path="isActive">
          <NSwitch :value="!!model.isActive" @update:value="v => (model.isActive = v)" />
        </NFormItem>
        <!-- 过期时间(下发 LiteLLM expires_at，过期请求由网关原生拒绝；清空=永不过期) -->
        <NFormItem :label="$t('page.gateway.aiKey.col.expiresAt')" path="expiresAt">
          <NDatePicker
            v-model:formatted-value="model.expiresAt"
            type="datetime"
            value-format="yyyy-MM-dd'T'HH:mm:ssXXX"
            clearable
            :placeholder="$t('page.gateway.aiKey.form.expiresAtPlaceholder')"
            class="w-full"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace :size="16">
          <NButton @click="closeDrawer">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped></style>
