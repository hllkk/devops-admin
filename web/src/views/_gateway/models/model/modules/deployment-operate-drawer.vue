<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { jsonClone } from '@sa/utils';
import { fetchCreateDeployment, fetchGetCredentialList, fetchGetProviderList, fetchUpdateDeployment } from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { BILLING_TYPE_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'DeploymentOperateDrawer' });

interface Props {
  /** the type of operation */
  operateType: NaiveUI.TableOperateType;
  /** the edit row data */
  rowData?: Api.Gateway.Deployment | null;
  /** 关联模型ID(新增隐含) */
  modelId?: CommonType.IdType | null;
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

const title = computed(() => (props.operateType === 'add' ? $t('page.gateway.deployment.add') : $t('page.gateway.deployment.edit')));

type Model = Api.Gateway.DeploymentOperateParams;

const model = ref<Model>(createDefaultModel());
/** litellmParams.model(厂商模型名，如 claude-sonnet-4-20250514) */
const vendorModel = ref('');

function createDefaultModel(): Model {
  return {
    deploymentId: null,
    modelId: props.modelId ?? null,
    credentialId: null,
    deployName: '',
    litellmParams: { model: '' },
    modelInfo: {},
    billingType: 'token',
    costPerCall: null,
    monthlyCallQuota: null,
    isActive: true
  };
}

const credentialOptions = ref<{ label: string; value: CommonType.IdType }[]>([]);
const providerOptions = ref<{ label: string; value: CommonType.IdType }[]>([]);
/** 表单临时供应商ID(仅过滤凭证下拉,不提交后端;deployment 经 credentialId 关联) */
const providerId = ref<CommonType.IdType | null>(null);

async function loadProviders() {
  const { data, error } = await fetchGetProviderList({
    pageNum: 1,
    pageSize: 100,
    name: null,
    providerType: null,
    isActive: null,
    params: {}
  });
  if (!error && data) {
    providerOptions.value = data.rows.map(p => ({ label: p.name, value: p.providerId }));
  }
}

async function loadCredentials(pid: CommonType.IdType | null) {
  const { data, error } = await fetchGetCredentialList({
    pageNum: 1,
    pageSize: 100,
    credentialName: null,
    providerId: pid,
    isActive: null,
    litellmSynced: null,
    params: {}
  });
  if (!error && data) {
    credentialOptions.value = data.rows.map(c => ({ label: c.credentialName, value: c.credentialId }));
  }
}

onMounted(() => {
  loadProviders();
  loadCredentials(null);
});

function handleProviderChange(val: CommonType.IdType | null) {
  model.value.credentialId = null;
  loadCredentials(val);
}

const billingTypeOptions = computed(() => BILLING_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

const rules: Record<'deployName', App.Global.FormRule> = {
  deployName: createRequiredRule($t('page.gateway.deployment.form.deployName.required'))
};

function handleUpdateModelWhenEdit() {
  model.value = createDefaultModel();
  vendorModel.value = '';
  providerId.value = null;

  if (props.operateType === 'edit' && props.rowData) {
    Object.assign(model.value, jsonClone(props.rowData));
    vendorModel.value = String(props.rowData.litellmParams?.model ?? '');
  }
}

function closeDrawer() {
  visible.value = false;
}

async function handleSubmit() {
  await validate();

  if (!vendorModel.value.trim()) {
    window.$message?.warning($t('page.gateway.deployment.form.vendorModel.required'));
    return;
  }

  // 保留 litellmParams 其他掩码键，仅覆盖 model
  model.value.litellmParams = { ...model.value.litellmParams, model: vendorModel.value.trim() };

  if (props.operateType === 'add') {
    const { error } = await fetchCreateDeployment(model.value);
    if (error) return;
    window.$message?.success($t('common.addSuccess'));
  }

  if (props.operateType === 'edit') {
    const { error } = await fetchUpdateDeployment(model.value);
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
      <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="120">
        <NFormItem :label="$t('page.gateway.deployment.col.provider')">
          <NSelect
            v-model:value="providerId"
            clearable
            filterable
            :options="providerOptions"
            :placeholder="$t('common.placeholderSelect')"
            @update:value="handleProviderChange"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.deployment.col.credential')" path="credentialId">
          <NSelect
            v-model:value="model.credentialId"
            clearable
            filterable
            :options="credentialOptions"
            :placeholder="$t('page.gateway.deployment.form.credentialPlaceholder')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.deployment.col.deployName')" path="deployName">
          <NInput v-model:value="model.deployName" :placeholder="$t('page.gateway.deployment.form.deployName.required')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.deployment.col.vendorModel')" path="vendorModel">
          <NInput v-model:value="vendorModel" :placeholder="$t('page.gateway.deployment.form.vendorModel.required')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.deployment.col.billingType')" path="billingType">
          <NSelect v-model:value="model.billingType" :options="billingTypeOptions" :placeholder="$t('common.placeholderSelect')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.deployment.col.costPerCall')" path="costPerCall">
          <NInputNumber v-model:value="model.costPerCall" :min="0" clearable :placeholder="$t('page.gateway.common.unlimited')" class="w-full">
            <template #suffix>¥</template>
          </NInputNumber>
        </NFormItem>
        <NFormItem :label="$t('page.gateway.deployment.col.monthlyCallQuota')" path="monthlyCallQuota">
          <NInputNumber v-model:value="model.monthlyCallQuota" :min="0" clearable :placeholder="$t('page.gateway.common.unlimited')" class="w-full" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.deployment.col.isActive')" path="isActive">
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
