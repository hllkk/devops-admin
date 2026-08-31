<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { fetchCreateBudgetRule, fetchUpdateBudgetRule } from '@/service/api/gateway';
import UserSelect from '@/components/custom/user-select.vue';
import { fetchGetDeptTree } from '@/service/api/system';

defineOptions({ name: 'BudgetRuleDrawer' });

interface Props {
  /** 编辑态数据(新增时为 null) */
  rowData?: Api.Gateway.BudgetRuleView | null;
  /** 维度锁定(dept/user);新增时必传,编辑时从 rowData 取 */
  scopeType?: 'dept' | 'user';
}

const props = defineProps<Props>();

interface Emits {
  (e: 'submitted'): void;
}

const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', { default: false });

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = ref<Api.Gateway.BudgetRuleOperateParams>(getDefaultModel());

function getDefaultModel(): Api.Gateway.BudgetRuleOperateParams {
  return {
    ruleId: null,
    scopeType: props.scopeType ?? 'dept',
    scopeId: null,
    budgetLimit: 0,
    budgetHardLimit: false,
    budgetDuration: '30d',
    softWarnPercent: 80,
    isActive: true
  };
}

const durationOptions = [
  { label: $t('page.gateway.budget.duration1d'), value: '1d' },
  { label: $t('page.gateway.budget.duration7d'), value: '7d' },
  { label: $t('page.gateway.budget.duration30d'), value: '30d' }
];

const deptOptions = ref<Api.Common.CommonTreeRecord>([]);
const deptLoaded = ref(false);

async function loadDeptTree() {
  if (deptLoaded.value) return;
  const { data, error } = await fetchGetDeptTree();
  if (!error && data) {
    deptOptions.value = data;
    deptLoaded.value = true;
  }
}

watch(visible, v => {
  if (v) {
    restoreValidation();
    if (props.rowData) {
      model.value = {
        ruleId: props.rowData.ruleId,
        scopeType: props.rowData.scopeType,
        scopeId: props.rowData.scopeId,
        budgetLimit: props.rowData.budgetLimit,
        budgetHardLimit: props.rowData.budgetHardLimit,
        budgetDuration: props.rowData.budgetDuration,
        softWarnPercent: props.rowData.softWarnPercent,
        isActive: props.rowData.isActive
      };
    } else {
      model.value = getDefaultModel();
    }
    if (model.value.scopeType === 'dept') loadDeptTree();
  }
});

const isEdit = computed(() => !!model.value.ruleId);
const title = computed(() => isEdit.value ? $t('page.gateway.budget.edit') : $t('page.gateway.budget.add'));

const rules: Record<string, App.Global.FormRule> = {
  scopeId: { type: 'number', required: true, trigger: ['blur', 'change'], message: $t('page.gateway.budget.form.scopeIdRequired') },
  budgetLimit: { type: 'number', required: true, trigger: ['blur', 'change'], message: $t('page.gateway.budget.form.limitRequired') }
};

const submitting = ref(false);

async function handleSubmit() {
  await validate();
  submitting.value = true;
  const fn = isEdit.value ? fetchUpdateBudgetRule : fetchCreateBudgetRule;
  const { error } = await fn(model.value);
  submitting.value = false;
  if (error) return;
  window.$message?.success(isEdit.value ? $t('common.updateSuccess') : $t('common.addSuccess'));
  visible.value = false;
  emit('submitted');
}
</script>

<template>
  <NModal v-model:show="visible" preset="card" :title="title" class="w-500px max-w-90%">
    <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="110">
      <NFormItem :label="$t('page.gateway.budget.scopeType')">
        <NRadioGroup v-model:value="model.scopeType" :disabled="isEdit">
          <NRadio value="dept">{{ $t('page.gateway.budget.scopeDept') }}</NRadio>
          <NRadio value="user">{{ $t('page.gateway.budget.scopeUser') }}</NRadio>
        </NRadioGroup>
      </NFormItem>
      <NFormItem v-if="model.scopeType === 'dept'" :label="$t('page.gateway.budget.scopeDept')" path="scopeId">
        <NTreeSelect
          v-model:value="model.scopeId"
          :disabled="isEdit"
          clearable
          filterable
          key-field="id"
          label-field="label"
          :options="deptOptions as []"
          :placeholder="$t('common.placeholderSelect')"
        />
      </NFormItem>
      <NFormItem v-else :label="$t('page.gateway.budget.scopeUser')" path="scopeId">
        <UserSelect
          v-model:value="model.scopeId"
          :disabled="isEdit"
          clearable
          filterable
          :placeholder="$t('common.placeholderSelect')"
        />
      </NFormItem>
      <NFormItem :label="$t('page.gateway.budget.budgetLimit')" path="budgetLimit">
        <NInputNumber v-model:value="model.budgetLimit" :min="0" :precision="2" class="w-full">
          <template #prefix>¥</template>
        </NInputNumber>
      </NFormItem>
      <NFormItem :label="$t('page.gateway.budget.duration')">
        <NRadioGroup v-model:value="model.budgetDuration">
          <NRadioButton v-for="opt in durationOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadioButton>
        </NRadioGroup>
      </NFormItem>
      <NFormItem :label="$t('page.gateway.budget.softWarnPercent')">
        <NInputNumber v-model:value="model.softWarnPercent" :min="1" :max="100" class="w-full">
          <template #suffix>%</template>
        </NInputNumber>
      </NFormItem>
      <NFormItem :label="$t('page.gateway.budget.hardLimit')">
        <NSwitch :value="!!model.budgetHardLimit" @update:value="(v: boolean) => (model.budgetHardLimit = v)" />
      </NFormItem>
      <NFormItem :label="$t('page.gateway.budget.isActive')">
        <NSwitch :value="!!model.isActive" @update:value="(v: boolean) => (model.isActive = v)" />
      </NFormItem>
    </NForm>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="visible = false">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped></style>
