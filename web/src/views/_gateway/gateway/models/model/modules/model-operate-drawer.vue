<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { jsonClone } from '@sa/utils';
import { fetchCreateModel, fetchUpdateModel } from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { MODEL_CATEGORY_OPTIONS, PROVIDER_TYPE_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'ModelOperateDrawer' });

interface Props {
  /** the type of operation */
  operateType: NaiveUI.TableOperateType;
  /** the edit row data */
  rowData?: Api.Gateway.Model | null;
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

const title = computed(() => (props.operateType === 'add' ? $t('page.gateway.model.add') : $t('page.gateway.model.edit')));

type Model = Api.Gateway.ModelOperateParams;

const model = ref<Model>(createDefaultModel());

function createDefaultModel(): Model {
  return {
    modelId: null,
    modelKey: '',
    name: '',
    category: 'chat',
    logoProviderType: '',
    capabilities: [],
    description: ''
  };
}

const categoryOptions = computed(() => MODEL_CATEGORY_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

const rules: Record<'modelKey' | 'name' | 'category', App.Global.FormRule> = {
  modelKey: createRequiredRule($t('page.gateway.model.form.modelKey.required')),
  name: createRequiredRule($t('page.gateway.model.form.name.required')),
  category: createRequiredRule($t('page.gateway.model.form.category.required'))
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
    const { error } = await fetchCreateModel(model.value);
    if (error) return;
    window.$message?.success($t('common.addSuccess'));
  }

  if (props.operateType === 'edit') {
    const { error } = await fetchUpdateModel(model.value);
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
        <NFormItem :label="$t('page.gateway.model.col.modelKey')" path="modelKey">
          <NInput v-model:value="model.modelKey" :placeholder="$t('page.gateway.model.form.modelKey.required')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.model.col.name')" path="name">
          <NInput v-model:value="model.name" :placeholder="$t('page.gateway.model.form.name.required')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.model.col.category')" path="category">
          <NSelect v-model:value="model.category" :options="categoryOptions" :placeholder="$t('common.placeholderSelect')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.model.col.logoProviderType')" path="logoProviderType">
          <NSelect
            v-model:value="model.logoProviderType"
            clearable
            filterable
            tag
            :options="PROVIDER_TYPE_OPTIONS"
            :placeholder="$t('common.placeholderSelect')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.model.col.capabilities')" path="capabilities">
          <NSelect
            v-model:value="model.capabilities"
            multiple
            filterable
            tag
            :options="[]"
            :placeholder="$t('page.gateway.model.form.capabilitiesPlaceholder')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.model.col.description')" path="description">
          <NInput v-model:value="model.description" type="textarea" :rows="3" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NAlert v-if="operateType === 'edit'" type="warning" :show-icon="true" class="mt-8px">
          {{ $t('page.gateway.model.form.renameTip') }}
        </NAlert>
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
