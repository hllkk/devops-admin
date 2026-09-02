<script setup lang="ts">
import { ref, watch } from 'vue';
import { fetchGetDeptTree, fetchGetUserSelect } from '@/service/api/system';
import { fetchGetMCPPublish, fetchPublishMCPServer } from '@/service/api/gateway';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({ name: 'MCPPublishDialog' });

interface Props {
  row: Api.Gateway.MCPServer | null;
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

type Model = Api.Gateway.MCPPublishParams;

const formModel = ref<Model>(createDefaultModel());
const loading = ref(false);

function createDefaultModel(): Model {
  return {
    mcpServerId: props.row?.mcpServerId ?? '',
    isPublished: props.row?.isPublished ?? false,
    visibilityType: props.row?.visibilityType ?? 'all',
    requiresApproval: props.row?.requiresApproval ?? false,
    departmentIds: [],
    userIds: []
  };
}

/** 指定部门/用户可见 + 发布时，对应列表必填；mixed 档两列表合计至少一项 */
const rules: Record<string, App.Global.FormRule> = {
  departmentIds: {
    trigger: ['change', 'blur'],
    validator: (_rule, value: CommonType.IdType[]) => {
      if (!formModel.value.isPublished) return true;
      if (formModel.value.visibilityType === 'selected' && (!value || value.length === 0)) {
        return new Error($t('page.gateway.mcp.publish.departmentRequired'));
      }
      if (
        formModel.value.visibilityType === 'mixed' &&
        (!value || value.length === 0) &&
        formModel.value.userIds.length === 0
      ) {
        return new Error($t('page.gateway.mcp.publish.mixedRequired'));
      }
      return true;
    }
  },
  userIds: {
    trigger: ['change', 'blur'],
    validator: (_rule, value: CommonType.IdType[]) => {
      if (!formModel.value.isPublished) return true;
      if (formModel.value.visibilityType === 'user' && (!value || value.length === 0)) {
        return new Error($t('page.gateway.mcp.publish.userRequired'));
      }
      if (
        formModel.value.visibilityType === 'mixed' &&
        (!value || value.length === 0) &&
        formModel.value.departmentIds.length === 0
      ) {
        return new Error($t('page.gateway.mcp.publish.mixedRequired'));
      }
      return true;
    }
  }
};

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
  if (!props.row?.mcpServerId) return;
  loading.value = true;
  const { data, error } = await fetchGetMCPPublish(props.row.mcpServerId);
  if (!error && data) {
    formModel.value = {
      mcpServerId: data.mcpServerId,
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
  formModel.value.visibilityType = val as 'all' | 'selected' | 'user' | 'mixed';
  // mixed 保留两侧已选(从单档切 mixed 追加另一侧)；切出档位清对应侧
  if (val !== 'selected' && val !== 'mixed') formModel.value.departmentIds = [];
  if (val !== 'user' && val !== 'mixed') formModel.value.userIds = [];
}

function closeDialog() {
  visible.value = false;
}

async function handleSubmit() {
  await validate();
  const { error } = await fetchPublishMCPServer(formModel.value);
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
    :title="$t('page.gateway.mcp.publish.title')"
    preset="card"
    class="w-560px max-w-90%"
    :mask-closable="false"
    content-style="max-height: calc(100vh - 220px); overflow-y: auto;"
  >
    <NForm ref="formRef" :model="formModel" :rules="rules" label-placement="left" :label-width="110">
      <p class="mb-12px text-12px text-slate-400">{{ $t('page.gateway.mcp.publish.subtitle') }}</p>
      <NFormItem :label="$t('page.gateway.mcp.publish.isPublished')">
        <div class="flex items-center gap-8px">
          <NSwitch v-model:value="formModel.isPublished" />
          <span class="text-12px text-slate-400">{{ $t('page.gateway.mcp.publish.autoGrantTip') }}</span>
        </div>
      </NFormItem>
      <template v-if="formModel.isPublished">
        <NFormItem :label="$t('page.gateway.mcp.publish.visibilityType')">
          <NRadioGroup :value="formModel.visibilityType" @update:value="handleVisibilityChange">
            <NRadio value="all">{{ $t('page.gateway.mcp.publish.visibilityAll') }}</NRadio>
            <NRadio value="selected">{{ $t('page.gateway.mcp.publish.visibilitySelected') }}</NRadio>
            <NRadio value="user">{{ $t('page.gateway.mcp.publish.visibilityUser') }}</NRadio>
            <NRadio value="mixed">{{ $t('page.gateway.mcp.publish.visibilityMixed') }}</NRadio>
          </NRadioGroup>
        </NFormItem>
        <NFormItem
          v-if="formModel.visibilityType === 'selected' || formModel.visibilityType === 'mixed'"
          :label="$t('page.gateway.mcp.publish.departmentIds')"
          path="departmentIds"
        >
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
        <NFormItem
          v-if="formModel.visibilityType === 'user' || formModel.visibilityType === 'mixed'"
          :label="$t('page.gateway.mcp.publish.userIds')"
          path="userIds"
        >
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
        <NFormItem :label="$t('page.gateway.mcp.publish.requiresApproval')">
          <div class="flex items-center gap-8px">
            <NSwitch v-model:value="formModel.requiresApproval" />
            <span class="text-12px text-slate-400">{{ $t('page.gateway.mcp.publish.requiresApprovalTip') }}</span>
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
