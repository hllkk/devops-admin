<script setup lang="tsx">
import { computed, ref, watch } from 'vue';
import { jsonClone } from '@sa/utils';
import { fetchCreateModel, fetchUpdateModel } from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { MODEL_CAPABILITY_PRESETS, MODEL_CATEGORY_OPTIONS, PROVIDER_TYPE_OPTIONS, getProviderIcon } from '@/constants/business/gateway';
import SvgIcon from '@/components/custom/svg-icon.vue';

defineOptions({ name: 'ModelOperateDrawer' });

interface Props {
  /** the type of operation */
  operateType: NaiveUI.TableOperateType;
  /** the edit row data */
  rowData?: Api.Gateway.Model | null;
}

const props = defineProps<Props>();

interface Emits {
  (e: 'submitted', modelId: CommonType.IdType | null): void;
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

/** 能力标签预设随类别联动；编辑回填不受切换清空影响 */
const capabilityOptions = computed(() =>
  (MODEL_CAPABILITY_PRESETS[model.value.category ?? ''] ?? []).map(c => ({ label: c, value: c }))
);

const rules: Record<'name' | 'category', App.Global.FormRule> = {
  name: createRequiredRule($t('page.gateway.model.form.name.required')),
  category: createRequiredRule($t('page.gateway.model.form.category.required'))
};

/** Logo 下拉选项渲染品牌图标 + 文本 */
function renderLogoLabel(option: { label?: string | number; value?: string | number }) {
  const value = String(option.value ?? '');
  return (
    <div class="flex items-center gap-8px">
      <SvgIcon localIcon={getProviderIcon(value)} class="h-18px w-18px" />
      <span>{String(option.label ?? value)}</span>
    </div>
  );
}

function handleCategoryChange() {
  model.value.capabilities = [];
}

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
    const { data, error } = await fetchCreateModel(model.value);
    if (error) return;
    window.$message?.success($t('common.addSuccess'));
    closeDrawer();
    emit('submitted', data?.modelId ?? null);
  }

  if (props.operateType === 'edit') {
    const { error } = await fetchUpdateModel(model.value);
    if (error) return;
    window.$message?.success($t('common.updateSuccess'));
    closeDrawer();
    emit('submitted', props.rowData?.modelId ?? null);
  }
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
        <NFormItem :label="$t('page.gateway.model.col.name')" path="name">
          <NInput v-model:value="model.name" :placeholder="$t('page.gateway.model.form.name.required')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.model.col.category')" path="category">
          <NSelect
            v-model:value="model.category"
            :options="categoryOptions"
            :placeholder="$t('common.placeholderSelect')"
            @update:value="handleCategoryChange"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.model.col.modelKey')" path="modelKey">
          <NInput v-model:value="model.modelKey" :placeholder="$t('page.gateway.model.form.modelKey.placeholder')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.model.col.logoProviderType')" path="logoProviderType">
          <NSelect
            v-model:value="model.logoProviderType"
            clearable
            filterable
            tag
            :options="PROVIDER_TYPE_OPTIONS"
            :render-label="renderLogoLabel"
            :placeholder="$t('common.placeholderSelect')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.model.col.capabilities')" path="capabilities">
          <NSelect
            v-model:value="model.capabilities"
            multiple
            filterable
            tag
            :options="capabilityOptions"
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
