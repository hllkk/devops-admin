<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { jsonClone } from '@sa/utils';
import { fetchCreateProvider, fetchUpdateProvider } from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { BILLING_TYPE_OPTIONS, CREDENTIAL_FORMAT_OPTIONS, PROVIDER_TYPE_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'ProviderOperateDrawer' });

interface Props {
  /** the type of operation */
  operateType: NaiveUI.TableOperateType;
  /** the edit row data */
  rowData?: Api.Gateway.Provider | null;
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

const title = computed(() => (props.operateType === 'add' ? $t('page.gateway.provider.add') : $t('page.gateway.provider.edit')));

type Model = Api.Gateway.ProviderOperateParams;

const model = ref<Model>(createDefaultModel());

function createDefaultModel(): Model {
  return {
    providerId: null,
    name: '',
    providerType: 'openai',
    billingType: 'token',
    monthlyBudget: null,
    isActive: true,
    description: '',
    supportedFormats: null
  };
}

const billingTypeOptions = computed(() => BILLING_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const formatOptions = computed(() => CREDENTIAL_FORMAT_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

const rules: Record<'name' | 'providerType' | 'billingType', App.Global.FormRule> = {
  name: createRequiredRule($t('page.gateway.provider.form.name.required')),
  providerType: createRequiredRule($t('page.gateway.provider.form.providerType.required')),
  billingType: createRequiredRule($t('page.gateway.provider.form.billingType.required'))
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
    const { error } = await fetchCreateProvider(model.value);
    if (error) return;
    window.$message?.success($t('common.addSuccess'));
  }

  if (props.operateType === 'edit') {
    const { error } = await fetchUpdateProvider(model.value);
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
  <NDrawer v-model:show="visible" :title="title" display-directive="show" :width="600" class="max-w-90%">
    <NDrawerContent :title="title" :native-scrollbar="false" closable>
      <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="100">
        <NFormItem :label="$t('page.gateway.provider.col.name')" path="name">
          <NInput v-model:value="model.name" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.provider.col.providerType')" path="providerType">
          <NSelect
            v-model:value="model.providerType"
            filterable
            tag
            :options="PROVIDER_TYPE_OPTIONS"
            :placeholder="$t('common.placeholderSelect')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.provider.col.supportedFormats')" path="supportedFormats">
          <NSelect
            v-model:value="model.supportedFormats"
            multiple
            :options="formatOptions"
            :placeholder="$t('common.placeholderSelect')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.provider.col.billingType')" path="billingType">
          <NSelect v-model:value="model.billingType" :options="billingTypeOptions" :placeholder="$t('common.placeholderSelect')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.provider.col.monthlyBudget')" path="monthlyBudget">
          <NInputNumber v-model:value="model.monthlyBudget" :min="0" clearable :placeholder="$t('page.gateway.common.unlimited')" class="w-full">
            <template #suffix>USD</template>
          </NInputNumber>
        </NFormItem>
        <NFormItem :label="$t('page.gateway.provider.col.isActive')" path="isActive">
          <NSwitch :value="!!model.isActive" @update:value="v => (model.isActive = v)" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.provider.col.description')" path="description">
          <NInput v-model:value="model.description" type="textarea" :rows="3" :placeholder="$t('common.placeholderInput')" />
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
