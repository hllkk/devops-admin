<script setup lang="ts">
import { ref, watch } from 'vue';
import { fetchGetDeptTree, fetchGetUserSelect } from '@/service/api/system';
import { fetchGetModelPublish, fetchPublishModel } from '@/service/api/gateway';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({ name: 'ModelPublishDialog' });

interface Props {
  /** 发布是模型级属性(多渠道共享) */
  model: Api.Gateway.Model;
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

type Model = Api.Gateway.ModelPublishParams;

const formModel = ref<Model>(createDefaultModel());

function createDefaultModel(): Model {
  return {
    modelId: props.model.modelId,
    isPublished: props.model.isPublished,
    visibilityType: props.model.visibilityType ?? 'all',
    requiresApproval: props.model.requiresApproval ?? false,
    departmentIds: [],
    userIds: []
  };
}

/** 指定部门/用户可见 + 发布时，对应列表必填 */
const rules: Record<string, App.Global.FormRule> = {
  departmentIds: {
    trigger: ['change', 'blur'],
    validator: (_rule, value: CommonType.IdType[]) => {
      if (formModel.value.isPublished && formModel.value.visibilityType === 'selected' && (!value || value.length === 0)) {
        return new Error($t('page.gateway.model.publish.departmentRequired'));
      }
      return true;
    }
  },
  userIds: {
    trigger: ['change', 'blur'],
    validator: (_rule, value: CommonType.IdType[]) => {
      if (formModel.value.isPublished && formModel.value.visibilityType === 'user' && (!value || value.length === 0)) {
        return new Error($t('page.gateway.model.publish.userRequired'));
      }
      return true;
    }
  }
};

const deptOptions = ref<Api.Common.CommonTreeRecord>([]);
const deptLoaded = ref(false);
const loading = ref(false);

async function loadDeptTree() {
  if (deptLoaded.value) return;
  const { data, error } = await fetchGetDeptTree();
  if (!error && data) {
    deptOptions.value = data;
    deptLoaded.value = true;
  }
}

const userOptions = ref<CommonType.Option<CommonType.IdType>[]>([]);
const userLoaded = ref(false);

async function loadUserOptions() {
  if (userLoaded.value) return;
  const { data, error } = await fetchGetUserSelect();
  if (!error && data) {
    userOptions.value = data.map(item => ({
      label: `${item.nickName} ( ${item.userName} )`,
      value: item.userId
    }));
    userLoaded.value = true;
  }
}

async function handleOpen() {
  formModel.value = createDefaultModel();
  restoreValidation();
  loadDeptTree();
  loadUserOptions();
  // 拉取权威发布设置(可见部门/用户投影行只在视图里返回)
  loading.value = true;
  const { data, error } = await fetchGetModelPublish(props.model.modelId!);
  if (!error && data) {
    formModel.value = {
      modelId: data.modelId,
      isPublished: data.isPublished,
      visibilityType: data.visibilityType,
      requiresApproval: data.requiresApproval,
      departmentIds: data.departmentIds ?? [],
      userIds: data.userIds ?? []
    };
  }
  loading.value = false;
}

function handleVisibilityChange(val: string) {
  formModel.value.visibilityType = val as 'all' | 'selected' | 'user';
  if (val !== 'selected') formModel.value.departmentIds = [];
  if (val !== 'user') formModel.value.userIds = [];
}

function closeDialog() {
  visible.value = false;
}

async function handleSubmit() {
  await validate();
  const { error } = await fetchPublishModel(formModel.value);
  if (error) return;
  window.$message?.success($t('common.updateSuccess'));
  closeDialog();
  emit('submitted');
}

watch(visible, val => {
  if (val) handleOpen();
});
</script>

<template>
  <NModal
    v-model:show="visible"
    :title="$t('page.gateway.model.publish.title')"
    preset="card"
    class="w-560px max-w-90%"
    :mask-closable="false"
    content-style="max-height: calc(100vh - 220px); overflow-y: auto;"
  >
    <NForm ref="formRef" :model="formModel" :rules="rules" label-placement="left" :label-width="110">
      <p class="mb-12px text-12px text-slate-400">{{ $t('page.gateway.model.publish.subtitle') }}</p>
      <NAlert v-if="formModel.isPublished && !formModel.requiresApproval && !model.modelKey" type="warning" :show-icon="true" class="mb-12px">
        {{ $t('page.gateway.model.publish.modelKeyUnsetTip') }}
      </NAlert>
      <NFormItem :label="$t('page.gateway.model.publish.isPublished')">
        <div class="flex items-center gap-8px">
          <NSwitch v-model:value="formModel.isPublished" />
          <span class="text-12px text-slate-400">{{ $t('page.gateway.model.publish.autoGrantTip') }}</span>
        </div>
      </NFormItem>
      <!-- 可见范围与领用审批仅发布开启时配置(参照 AIHelms 发布设置弹窗) -->
      <template v-if="formModel.isPublished">
        <NFormItem :label="$t('page.gateway.model.publish.visibilityType')">
          <NRadioGroup :value="formModel.visibilityType" @update:value="handleVisibilityChange">
            <NRadio value="all">{{ $t('page.gateway.model.publish.visibilityAll') }}</NRadio>
            <NRadio value="selected">{{ $t('page.gateway.model.publish.visibilitySelected') }}</NRadio>
            <NRadio value="user">{{ $t('page.gateway.model.publish.visibilityUser') }}</NRadio>
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="formModel.visibilityType === 'selected'" :label="$t('page.gateway.model.publish.departmentIds')" path="departmentIds">
          <NTreeSelect
            v-model:value="formModel.departmentIds"
            multiple
            checkable
            filterable
            :loading="loading || !deptLoaded"
            key-field="id"
            label-field="label"
            :options="deptOptions as []"
            :placeholder="$t('common.placeholderSelect')"
          />
        </NFormItem>
        <NFormItem v-if="formModel.visibilityType === 'user'" :label="$t('page.gateway.model.publish.userIds')" path="userIds">
          <NSelect
            v-model:value="formModel.userIds"
            multiple
            filterable
            clearable
            :loading="loading || !userLoaded"
            :options="userOptions"
            :placeholder="$t('common.placeholderSelect')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.model.publish.requiresApproval')">
          <div class="flex items-center gap-8px">
            <NSwitch v-model:value="formModel.requiresApproval" />
            <span class="text-12px text-slate-400">{{ $t('page.gateway.model.publish.requiresApprovalTip') }}</span>
          </div>
        </NFormItem>
      </template>
    </NForm>
    <template #footer>
      <NSpace justify="end" :size="12">
        <NButton size="small" @click="closeDialog">{{ $t('common.cancel') }}</NButton>
        <NButton size="small" type="primary" :loading="loading" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped></style>
