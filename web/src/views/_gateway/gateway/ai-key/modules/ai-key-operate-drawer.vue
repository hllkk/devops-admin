<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { jsonClone } from '@sa/utils';
import { fetchCreateAiKey, fetchGetAvailableModels, fetchUpdateAiKey } from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import {
  BUDGET_DURATION_OPTIONS,
  KEY_TYPE_OPTIONS,
  OWNER_TYPE_OPTIONS,
  RATE_LIMIT_MODE_OPTIONS
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
    models: [],
    modelBudgets: {},
    budgetLimit: null,
    budgetHardLimit: false,
    budgetDuration: '30d',
    rateLimitMode: 'none',
    tpmLimit: null,
    rpmLimit: null,
    modelLimits: {},
    isActive: true
  };
}

const modelOptions = ref<{ label: string; value: string }[]>([]);

async function loadModels() {
  const { data, error } = await fetchGetAvailableModels();
  if (!error && data) {
    modelOptions.value = data.map(m => ({ label: m.name, value: m.modelKey }));
  }
}

onMounted(loadModels);

const keyTypeOptions = computed(() => KEY_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const ownerTypeOptions = computed(() => OWNER_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const rateLimitModeOptions = computed(() => RATE_LIMIT_MODE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const budgetDurationOptions = computed(() => BUDGET_DURATION_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

/** ownerId 为 IdType(可能含 string)，NInputNumber 要 number，此处桥接 */
const ownerIdNumber = computed<number | null>({
  get: () => (model.value.ownerId as number | null) ?? null,
  set: v => {
    model.value.ownerId = v;
  }
});

const rules: Record<'keyType' | 'ownerType' | 'ownerId', App.Global.FormRule> = {
  keyType: createRequiredRule($t('page.gateway.aiKey.form.keyType.required')),
  ownerType: createRequiredRule($t('page.gateway.aiKey.form.ownerType.required')),
  ownerId: createRequiredRule($t('page.gateway.aiKey.form.ownerId.required'))
};

function handleUpdateModelWhenEdit() {
  model.value = createDefaultModel();

  if (props.operateType === 'edit' && props.rowData) {
    Object.assign(model.value, jsonClone(props.rowData));
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
        <!-- 归属信息(创建必填且不可改) -->
        <NFormItem :label="$t('page.gateway.aiKey.col.keyType')" path="keyType">
          <NSelect v-model:value="model.keyType" :disabled="operateType === 'edit'" :options="keyTypeOptions" :placeholder="$t('common.placeholderSelect')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.ownerType')" path="ownerType">
          <NSelect v-model:value="model.ownerType" :disabled="operateType === 'edit'" :options="ownerTypeOptions" :placeholder="$t('common.placeholderSelect')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.ownerId')" path="ownerId">
          <NInputNumber v-model:value="ownerIdNumber" :disabled="operateType === 'edit'" :placeholder="$t('page.gateway.aiKey.form.ownerId.required')" class="w-full" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.name')" path="name">
          <NInput v-model:value="model.name" :placeholder="$t('page.gateway.aiKey.form.namePlaceholder')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.description')" path="description">
          <NInput v-model:value="model.description" type="textarea" :rows="2" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>

        <!-- 授权模型 -->
        <NFormItem :label="$t('page.gateway.aiKey.col.models')" path="models">
          <NSelect
            v-model:value="model.models"
            multiple
            filterable
            :options="modelOptions"
            :placeholder="$t('page.gateway.aiKey.form.modelsPlaceholder')"
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
          <NSwitch :value="!!model.budgetHardLimit" @update:value="v => (model.budgetHardLimit = v)" />
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
