<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { jsonClone } from '@sa/utils';
import {
  fetchCreateDeployment,
  fetchGetCredentialList,
  fetchGetProviderList,
  fetchUpdateDeployment,
  fetchUpdateModel
} from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { BILLING_TYPE_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'DeploymentOperateDialog' });

interface Props {
  /** the type of operation */
  operateType: NaiveUI.TableOperateType;
  /** the edit row data */
  rowData?: Api.Gateway.Deployment | null;
  /** 关联模型(新增时若其 modelKey 为空，需在本表单补填模型 ID) */
  model?: Api.Gateway.Model | null;
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

const formModel = ref<Model>(createDefaultModel());
/** litellmParams 非空引用(createDefault 已给完整对象，编辑回填覆盖后仍非空) */
const params = computed(() => formModel.value.litellmParams!);
/** litellmParams.model(厂商模型名，如 claude-sonnet-4-20250514) */
const vendorModel = ref('');
/** 关联模型未设置模型 ID 时的新增部署补填值(创建部署前先回写模型) */
const routeKey = ref('');
/** 折叠分组展开项(编辑回填时已有值则自动展开对应组) */
const expandedNames = ref<string[]>([]);

/** 新增部署且关联模型缺模型 ID 时，表单顶部补填(部署路由组名锚点，后端强制要求) */
const needRouteKey = computed(() => props.operateType === 'add' && !!props.model && !props.model.modelKey);
/** 计费与配额组仅按次/包月计费时展示 */
const showBillingGroup = computed(() => formModel.value.billingType !== 'token');
/** 月配额仅包月计费有意义 */
const showMonthlyQuota = computed(() => formModel.value.billingType === 'monthly_quota');

function createDefaultModel(): Model {
  return {
    deploymentId: null,
    modelId: props.model?.modelId ?? null,
    credentialId: null,
    deployName: '',
    litellmParams: { model: '', tags: [], use_in_pass_through: false, drop_params: false },
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
  formModel.value.credentialId = null;
  loadCredentials(val);
}

const billingTypeOptions = computed(() => BILLING_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

const rules: Record<'deployName', App.Global.FormRule> = {
  deployName: createRequiredRule($t('page.gateway.deployment.form.deployName.required'))
};

function handleUpdateModelWhenEdit() {
  formModel.value = createDefaultModel();
  vendorModel.value = '';
  routeKey.value = '';
  providerId.value = null;

  if (props.operateType === 'edit' && props.rowData) {
    Object.assign(formModel.value, jsonClone(props.rowData));
    vendorModel.value = String(props.rowData.litellmParams?.model ?? '');
    if (formModel.value.litellmParams == null) {
      formModel.value.litellmParams = { model: '', tags: [], use_in_pass_through: false, drop_params: false };
    }
    // 路由参数键 normalize(缺省给安全形态，不动其它掩码键)
    const p = params.value;
    p.tags = Array.isArray(p.tags) ? p.tags : [];
    p.use_in_pass_through = p.use_in_pass_through === true;
    p.drop_params = p.drop_params === true;
    // 已有值则自动展开对应折叠组
    expandedNames.value = computeExpandedNames();
  } else {
    expandedNames.value = [];
  }
}

function computeExpandedNames(): string[] {
  const names: string[] = [];
  const p: Record<string, any> = params.value ?? {};
  if (
    p.weight != null ||
    p.order != null ||
    p.timeout != null ||
    p.stream_timeout != null ||
    p.max_retries != null ||
    (Array.isArray(p.tags) && p.tags.length > 0)
  ) {
    names.push('routing');
  }
  if (formModel.value.costPerCall != null || formModel.value.monthlyCallQuota != null) {
    names.push('billing');
  }
  if (p.use_in_pass_through || p.drop_params) {
    names.push('advanced');
  }
  return names;
}

function closeDialog() {
  visible.value = false;
}

async function handleSubmit() {
  await validate();

  if (!vendorModel.value.trim()) {
    window.$message?.warning($t('page.gateway.deployment.form.vendorModel.required'));
    return;
  }

  if (needRouteKey.value) {
    if (!routeKey.value.trim()) {
      window.$message?.warning($t('page.gateway.deployment.form.modelKey.required'));
      return;
    }
    // 壳模型先回写模型 ID(部署路由组名锚点)，再创建部署
    const { error } = await fetchUpdateModel({
      modelId: props.model!.modelId,
      modelKey: routeKey.value.trim(),
      name: props.model!.name,
      category: props.model!.category,
      logoProviderType: props.model!.logoProviderType,
      capabilities: props.model!.capabilities ?? [],
      description: props.model!.description ?? ''
    });
    if (error) return;
  }

  // 保留 litellmParams 其他掩码键，仅覆盖 model + 剔除清空的可选路由键
  const nextParams: Record<string, any> = { ...formModel.value.litellmParams, model: vendorModel.value.trim() };
  for (const key of ['weight', 'order', 'timeout', 'stream_timeout', 'max_retries', 'tags']) {
    const v = nextParams[key];
    if (v === null || v === undefined || v === '' || (Array.isArray(v) && v.length === 0)) {
      Reflect.deleteProperty(nextParams, key);
    }
  }
  nextParams.use_in_pass_through = nextParams.use_in_pass_through === true;
  nextParams.drop_params = nextParams.drop_params === true;
  formModel.value.litellmParams = nextParams;

  if (props.operateType === 'add') {
    const { error } = await fetchCreateDeployment(formModel.value);
    if (error) return;
    window.$message?.success($t('common.addSuccess'));
  }

  if (props.operateType === 'edit') {
    const { error } = await fetchUpdateDeployment(formModel.value);
    if (error) return;
    window.$message?.success($t('common.updateSuccess'));
  }

  closeDialog();
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
  <NModal
    v-model:show="visible"
    :title="title"
    preset="card"
    class="w-640px max-w-90%"
    :mask-closable="false"
    content-style="max-height: calc(100vh - 220px); overflow-y: auto;"
  >
    <NForm ref="formRef" :model="formModel" :rules="rules" label-placement="left" :label-width="120">
      <template v-if="needRouteKey">
        <NAlert type="warning" :show-icon="true" class="mb-12px">
          {{ $t('page.gateway.deployment.form.modelKey.tip') }}
        </NAlert>
        <NFormItem :label="$t('page.gateway.model.col.modelKey')">
          <NInput v-model:value="routeKey" :placeholder="$t('page.gateway.deployment.form.modelKey.required')" />
        </NFormItem>
      </template>

      <!-- 核心配置 -->
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
          v-model:value="formModel.credentialId"
          clearable
          filterable
          :options="credentialOptions"
          :placeholder="$t('page.gateway.deployment.form.credentialPlaceholder')"
        />
      </NFormItem>
      <NFormItem :label="$t('page.gateway.deployment.col.deployName')" path="deployName">
        <NInput v-model:value="formModel.deployName" :placeholder="$t('page.gateway.deployment.form.deployName.required')" />
      </NFormItem>
      <NFormItem :label="$t('page.gateway.deployment.col.vendorModel')">
        <div class="w-full">
          <NInput v-model:value="vendorModel" :placeholder="$t('page.gateway.deployment.form.vendorModel.required')" />
          <p class="mt-4px text-12px text-slate-400">{{ $t('page.gateway.deployment.form.vendorModelTip') }}</p>
        </div>
      </NFormItem>
      <NFormItem :label="$t('page.gateway.deployment.col.billingType')" path="billingType">
        <NSelect v-model:value="formModel.billingType" :options="billingTypeOptions" :placeholder="$t('common.placeholderSelect')" />
      </NFormItem>

      <!-- 分组折叠：计费与配额 / 路由配置 / 高级设置 -->
      <NCollapse v-model:expanded-names="expandedNames" class="mt-4px">
        <NCollapseItem v-if="showBillingGroup" :title="$t('page.gateway.deployment.group.billing')" name="billing">
          <NGrid responsive="screen" item-responsive :x-gap="16">
            <NFormItemGi span="24 s:12" :label="$t('page.gateway.deployment.col.costPerCall')" path="costPerCall">
              <NInputNumber v-model:value="formModel.costPerCall" :min="0" clearable :placeholder="$t('page.gateway.common.unlimited')" class="w-full">
                <template #suffix>¥</template>
              </NInputNumber>
            </NFormItemGi>
            <NFormItemGi v-if="showMonthlyQuota" span="24 s:12" :label="$t('page.gateway.deployment.col.monthlyCallQuota')" path="monthlyCallQuota">
              <NInputNumber v-model:value="formModel.monthlyCallQuota" :min="0" clearable :placeholder="$t('page.gateway.common.unlimited')" class="w-full" />
            </NFormItemGi>
          </NGrid>
        </NCollapseItem>

        <NCollapseItem :title="$t('page.gateway.deployment.group.routing')" name="routing">
          <p class="mb-8px text-12px text-slate-400">{{ $t('page.gateway.deployment.form.routingTip') }}</p>
          <NGrid responsive="screen" item-responsive :x-gap="16">
            <NFormItemGi span="24 s:12" :label="$t('page.gateway.deployment.col.weight')">
              <NInputNumber
                v-model:value="params.weight"
                :min="0"
                clearable
                :placeholder="$t('page.gateway.deployment.form.weightPlaceholder')"
                class="w-full"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12" :label="$t('page.gateway.deployment.col.order')">
              <NInputNumber
                v-model:value="params.order"
                :min="0"
                clearable
                :placeholder="$t('page.gateway.deployment.form.orderPlaceholder')"
                class="w-full"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12" :label="$t('page.gateway.deployment.col.timeout')">
              <NInputNumber
                v-model:value="params.timeout"
                :min="0"
                clearable
                :placeholder="$t('page.gateway.deployment.form.timeoutPlaceholder')"
                class="w-full"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12" :label="$t('page.gateway.deployment.col.streamTimeout')">
              <NInputNumber
                v-model:value="params.stream_timeout"
                :min="0"
                clearable
                :placeholder="$t('page.gateway.deployment.form.streamTimeoutPlaceholder')"
                class="w-full"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12" :label="$t('page.gateway.deployment.col.maxRetries')">
              <NInputNumber
                v-model:value="params.max_retries"
                :min="0"
                clearable
                :placeholder="$t('page.gateway.deployment.form.maxRetriesPlaceholder')"
                class="w-full"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12" :label="$t('page.gateway.deployment.col.tags')">
              <NSelect
                v-model:value="params.tags"
                multiple
                filterable
                tag
                :options="[]"
                :placeholder="$t('page.gateway.deployment.form.tagsPlaceholder')"
              />
            </NFormItemGi>
          </NGrid>
        </NCollapseItem>

        <NCollapseItem :title="$t('page.gateway.deployment.group.advanced')" name="advanced">
          <NFormItem :label="$t('page.gateway.deployment.col.useInPassThrough')">
            <NSwitch :value="params.use_in_pass_through === true" @update:value="v => (params.use_in_pass_through = v)" />
            <span v-if="params.use_in_pass_through" class="ml-8px text-12px text-amber-500">
              {{ $t('page.gateway.deployment.form.passThroughTip') }}
            </span>
          </NFormItem>
          <NFormItem :label="$t('page.gateway.deployment.col.dropParams')">
            <NSwitch :value="params.drop_params === true" @update:value="v => (params.drop_params = v)" />
          </NFormItem>
        </NCollapseItem>
      </NCollapse>

      <NFormItem :label="$t('page.gateway.deployment.col.isActive')" path="isActive" class="mt-8px">
        <NSwitch :value="!!formModel.isActive" @update:value="v => (formModel.isActive = v)" />
      </NFormItem>
    </NForm>
    <template #footer>
      <NSpace :size="16" justify="end">
        <NButton @click="closeDialog">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped></style>
