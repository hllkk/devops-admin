<script setup lang="ts">
import { ref, watch } from 'vue';
import { fetchGetDeptTree, fetchGetUserSelect } from '@/service/api/system';
import { fetchGetSkillPublish, fetchPublishSkill } from '@/service/api/gateway';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({ name: 'SkillPublishDialog' });

interface Props {
  row: Api.Gateway.Skill | null;
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

type Model = Api.Gateway.SkillPublishParams;

const formModel = ref<Model>(createDefaultModel());
const loading = ref(false);

function createDefaultModel(): Model {
  return {
    skillId: props.row?.skillId ?? '',
    isPublished: props.row?.isPublished ?? false,
    visibilityType: props.row?.visibilityType ?? 'all',
    requiresApproval: props.row?.requiresApproval ?? false,
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
        return new Error($t('page.gateway.skill.publish.departmentRequired'));
      }
      return true;
    }
  },
  userIds: {
    trigger: ['change', 'blur'],
    validator: (_rule, value: CommonType.IdType[]) => {
      if (formModel.value.isPublished && formModel.value.visibilityType === 'user' && (!value || value.length === 0)) {
        return new Error($t('page.gateway.skill.publish.userRequired'));
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
  if (!props.row?.skillId) return;
  loading.value = true;
  const { data, error } = await fetchGetSkillPublish(props.row.skillId);
  if (!error && data) {
    formModel.value = {
      skillId: data.skillId,
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
  if (formModel.value.isPublished && !props.row?.zipFilename) {
    window.$message?.warning($t('page.gateway.skill.publish.needPackage'));
    return;
  }
  const { error } = await fetchPublishSkill(formModel.value);
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
    :title="$t('page.gateway.skill.publish.title')"
    preset="card"
    class="w-560px max-w-90%"
    :mask-closable="false"
    content-style="max-height: calc(100vh - 220px); overflow-y: auto;"
  >
    <NForm ref="formRef" :model="formModel" :rules="rules" label-placement="left" :label-width="110">
      <p class="mb-12px text-12px text-slate-400">{{ $t('page.gateway.skill.publish.subtitle') }}</p>
      <NFormItem :label="$t('page.gateway.skill.publish.isPublished')">
        <div class="flex items-center gap-8px">
          <NSwitch v-model:value="formModel.isPublished" />
          <span class="text-12px text-slate-400">{{ $t('page.gateway.skill.publish.autoGrantTip') }}</span>
        </div>
      </NFormItem>
      <template v-if="formModel.isPublished">
        <NFormItem :label="$t('page.gateway.skill.publish.visibilityType')">
          <NRadioGroup :value="formModel.visibilityType" @update:value="handleVisibilityChange">
            <NRadio value="all">{{ $t('page.gateway.skill.publish.visibilityAll') }}</NRadio>
            <NRadio value="selected">{{ $t('page.gateway.skill.publish.visibilitySelected') }}</NRadio>
            <NRadio value="user">{{ $t('page.gateway.skill.publish.visibilityUser') }}</NRadio>
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="formModel.visibilityType === 'selected'" :label="$t('page.gateway.skill.publish.departmentIds')" path="departmentIds">
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
        <NFormItem v-if="formModel.visibilityType === 'user'" :label="$t('page.gateway.skill.publish.userIds')" path="userIds">
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
        <NFormItem :label="$t('page.gateway.skill.publish.requiresApproval')">
          <div class="flex items-center gap-8px">
            <NSwitch v-model:value="formModel.requiresApproval" />
            <span class="text-12px text-slate-400">{{ $t('page.gateway.skill.publish.requiresApprovalTip') }}</span>
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
