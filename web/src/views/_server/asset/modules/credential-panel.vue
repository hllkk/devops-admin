<script setup lang="tsx">
import { computed, ref } from 'vue';
import { NTag, NTime } from 'naive-ui';
import {
  fetchBatchDeleteCredentials,
  fetchCreateCredential,
  fetchGetCredentialList,
  fetchUpdateCredential
} from '@/service/api/server';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { ACTIVE_OPTIONS, CREDENTIAL_FORM_FIELDS, CREDENTIAL_TYPE_OPTIONS } from '@/constants/business/server';
import ButtonIcon from '@/components/custom/button-icon.vue';

defineOptions({ name: 'ServerCredentialPanel' });

const appStore = useAppStore();

const searchParams = ref<Api.Server.CredentialSearchParams>({
  pageNum: 1,
  pageSize: 10,
  credentialName: null,
  credentialType: null,
  isActive: null,
  params: {}
});

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => fetchGetCredentialList(searchParams.value),
  transform: response => defaultTransform(response),
  onPaginationParamsChange: params => {
    searchParams.value.pageNum = params.page;
    searchParams.value.pageSize = params.pageSize;
  },
  columns: () => [
    {
      key: 'credentialName',
      title: $t('page.server.credential.col.credentialName'),
      align: 'left',
      minWidth: 160,
      ellipsis: { tooltip: true }
    },
    {
      key: 'credentialType',
      title: $t('page.server.credential.col.credentialType'),
      align: 'center',
      minWidth: 100,
      render: row => <NTag size="small">{credentialTypeLabel(row.credentialType)}</NTag>
    },
    {
      key: 'credentialValues',
      title: $t('page.server.credential.col.credentialValues'),
      align: 'center',
      minWidth: 200,
      render: row => {
        const entries = Object.entries(row.credentialValues ?? {});
        if (entries.length === 0) return <span class="text-slate-400">-</span>;
        return (
          <div class="flex flex-wrap justify-center gap-4px">
            {entries.map(([k, v]) => (
              <NTag size="small" bordered={false} class="font-mono">
                {k}={v}
              </NTag>
            ))}
          </div>
        );
      }
    },
    {
      key: 'description',
      title: $t('page.server.credential.col.description'),
      align: 'center',
      minWidth: 160,
      ellipsis: { tooltip: true },
      render: row => row.description || <span class="text-slate-400">-</span>
    },
    {
      key: 'isActive',
      title: $t('page.server.credential.col.isActive'),
      align: 'center',
      minWidth: 80,
      render: row => <NTag type={row.isActive ? 'success' : 'default'}>{$t(row.isActive ? 'page.server.common.active' : 'page.server.common.inactive')}</NTag>
    },
    {
      key: 'updateTime',
      title: $t('page.server.common.updateTime'),
      align: 'center',
      minWidth: 170,
      render: row => <NTime time={Date.parse(row.updateTime)} format="yyyy-MM-dd HH:mm:ss" />
    },
    {
      key: 'operate',
      title: $t('common.operate'),
      align: 'center',
      width: 140,
      render: row => (
        <div class="flex-center gap-8px">
          <ButtonIcon
            text
            type="primary"
            icon="material-symbols:drive-file-rename-outline-outline"
            tooltipContent={$t('common.edit')}
            onClick={() => handleEdit(row)}
          />
          <ButtonIcon
            text
            type="error"
            icon="material-symbols:delete-outline"
            tooltipContent={$t('common.delete')}
            popconfirmContent={$t('common.confirmDelete')}
            onPositiveClick={() => handleDelete(row.credentialId!)}
          />
        </div>
      )
    }
  ]
});

const { checkedRowKeys, onBatchDeleted, onDeleted } = useTableOperate(data, 'credentialId', getData);

const credentialTypeOptions = computed(() => CREDENTIAL_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

function credentialTypeLabel(v: string): string {
  return credentialTypeOptions.value.find(o => o.value === v)?.label ?? v;
}

const activeOptions = computed(() => ACTIVE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

const isActiveSearch = computed<number | null>({
  get: () => (searchParams.value.isActive === null ? null : searchParams.value.isActive ? 1 : 0),
  set: v => {
    searchParams.value.isActive = v === null ? null : v === 1;
  }
});

function resetSearch() {
  searchParams.value = {
    pageNum: 1,
    pageSize: searchParams.value.pageSize,
    credentialName: null,
    credentialType: null,
    isActive: null,
    params: {}
  };
  getDataByPage();
}

// ── 新增/编辑弹窗 ──
type CredentialModel = Api.Server.CredentialOperateParams;

const showModal = ref(false);
const submitLoading = ref(false);
const editingCredential = ref<Api.Server.Credential | null>(null);
const credentialModel = ref<CredentialModel>(createDefaultCredentialModel());

const { formRef, validate, restoreValidation } = useNaiveForm();
const { createRequiredRule } = useFormRules();

const rules: Record<'credentialName' | 'credentialType', App.Global.FormRule> = {
  credentialName: createRequiredRule($t('page.server.credential.form.nameRequired')),
  credentialType: createRequiredRule($t('page.server.credential.form.typeRequired'))
};

const modalTitle = computed(() => (editingCredential.value ? $t('page.server.credential.edit') : $t('page.server.credential.add')));

// 当前类型的表单字段模板(类型建后不可改,编辑态锁定)
const formFields = computed(() =>
  (credentialModel.value.credentialType ? CREDENTIAL_FORM_FIELDS[credentialModel.value.credentialType] : []).map(f => ({
    ...f,
    label: $t(f.label)
  }))
);

function createDefaultCredentialModel(): CredentialModel {
  return {
    credentialId: null,
    credentialName: '',
    credentialType: null,
    credentialValues: {},
    description: '',
    isActive: true
  };
}

function handleAdd() {
  editingCredential.value = null;
  credentialModel.value = createDefaultCredentialModel();
  showModal.value = true;
  restoreValidation();
}

function handleEdit(row: Api.Server.Credential) {
  editingCredential.value = row;
  // credentialValues 掩码原样回填(未修改的敏感值回传掩码串=后端保留旧明文)
  credentialModel.value = {
    credentialId: row.credentialId,
    credentialName: row.credentialName,
    credentialType: row.credentialType,
    credentialValues: { ...row.credentialValues },
    description: row.description,
    isActive: row.isActive
  };
  showModal.value = true;
  restoreValidation();
}

async function handleSubmit() {
  await validate();
  if (!credentialModel.value.credentialType) return;
  // 按类型模板过滤提交键(避免残留其它类型字段)
  const fields = CREDENTIAL_FORM_FIELDS[credentialModel.value.credentialType];
  const values: Record<string, string> = {};
  for (const f of fields) {
    const v = credentialModel.value.credentialValues[f.key];
    if (v !== undefined && v !== '') values[f.key] = v;
  }
  // 编辑态至少保留一个键(全掩码回传也算未改,后端合并旧明文)
  if (Object.keys(values).length === 0) {
    window.$message?.warning($t('page.server.credential.form.valuesRequired'));
    return;
  }
  submitLoading.value = true;
  const payload: Api.Server.CredentialOperateParams = { ...credentialModel.value, credentialValues: values };
  const isEdit = !!editingCredential.value;
  const { error } = isEdit ? await fetchUpdateCredential(payload) : await fetchCreateCredential(payload);
  submitLoading.value = false;
  if (error) return;
  window.$message?.success($t(isEdit ? 'common.updateSuccess' : 'common.addSuccess'));
  showModal.value = false;
  getData();
  notifyChanged();
}

async function handleDelete(credentialId: CommonType.IdType) {
  const { error } = await fetchBatchDeleteCredentials([credentialId]);
  if (error) return;
  onDeleted();
}

async function handleBatchDelete() {
  const { error } = await fetchBatchDeleteCredentials(checkedRowKeys.value);
  if (error) return;
  onBatchDeleted();
}

// 凭据增/删后资产表单的凭据下拉需刷新——由父组件监听 changed 事件处理
const emit = defineEmits<{
  (e: 'changed'): void;
}>();

function notifyChanged() {
  emit('changed');
}
</script>

<template>
  <div class="h-full flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <NCard :bordered="false" size="small" class="card-wrapper">
      <NCollapse>
        <NCollapseItem :title="$t('common.search')" name="credential-search">
          <NForm label-placement="left" :label-width="80">
            <NGrid responsive="screen" item-responsive>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.server.credential.col.credentialName')" class="pr-24px">
                <NInput v-model:value="searchParams.credentialName" clearable :placeholder="$t('common.keywordSearch')" @keyup.enter="() => getDataByPage()" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.server.credential.col.credentialType')" class="pr-24px">
                <NSelect v-model:value="searchParams.credentialType" clearable :options="credentialTypeOptions" :placeholder="$t('common.placeholderSelect')" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.server.credential.col.isActive')" class="pr-24px">
                <NSelect v-model:value="isActiveSearch" clearable :options="activeOptions" :placeholder="$t('common.placeholderSelect')" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:16" class="pr-24px">
                <NSpace class="w-full" justify="end">
                  <NButton @click="resetSearch">
                    <template #icon>
                      <icon-ic-round-refresh class="text-icon" />
                    </template>
                    {{ $t('common.reset') }}
                  </NButton>
                  <NButton type="primary" ghost @click="() => getDataByPage()">
                    <template #icon>
                      <icon-ic-round-search class="text-icon" />
                    </template>
                    {{ $t('common.search') }}
                  </NButton>
                </NSpace>
              </NFormItemGi>
            </NGrid>
          </NForm>
        </NCollapseItem>
      </NCollapse>
    </NCard>
    <NCard :title="$t('page.server.credential.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
      <template #header-extra>
        <TableHeaderOperation
          v-model:columns="columnChecks"
          :disabled-delete="checkedRowKeys.length === 0"
          :loading="loading"
          :show-add="true"
          :show-delete="true"
          @add="handleAdd"
          @delete="handleBatchDelete"
          @refresh="getData"
        />
      </template>
      <NDataTable
        v-model:checked-row-keys="checkedRowKeys"
        :columns="columns"
        :data="data"
        size="small"
        :flex-height="!appStore.isMobile"
        :scroll-x="scrollX"
        :loading="loading"
        remote
        :row-key="row => row.credentialId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
    </NCard>
    <NModal v-model:show="showModal" preset="card" :title="modalTitle" class="w-520px">
      <NForm ref="formRef" :model="credentialModel" :rules="rules" label-placement="left" :label-width="90">
        <NFormItem :label="$t('page.server.credential.col.credentialName')" path="credentialName">
          <NInput v-model:value="credentialModel.credentialName" :placeholder="$t('page.server.credential.form.namePlaceholder')" />
        </NFormItem>
        <NFormItem :label="$t('page.server.credential.col.credentialType')" path="credentialType">
          <NSelect
            v-model:value="credentialModel.credentialType"
            :options="credentialTypeOptions"
            :disabled="!!editingCredential"
            :placeholder="$t('page.server.credential.form.typePlaceholder')"
          />
        </NFormItem>
        <NFormItem v-for="f in formFields" :key="f.key" :label="f.label">
          <NInput
            v-model:value="credentialModel.credentialValues[f.key]"
            :type="f.sensitive ? 'password' : 'text'"
            show-password-on="click"
            :placeholder="f.sensitive ? $t('page.server.credential.form.sensitivePlaceholder') : f.placeholder"
          />
        </NFormItem>
        <NFormItem :label="$t('page.server.credential.col.description')" path="description">
          <NInput v-model:value="credentialModel.description" type="textarea" :rows="2" :placeholder="$t('page.server.credential.form.descPlaceholder')" />
        </NFormItem>
        <NFormItem :label="$t('page.server.credential.col.isActive')" path="isActive">
          <NSwitch :value="!!credentialModel.isActive" @update:value="v => (credentialModel.isActive = v)" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace class="w-full" justify="end">
          <NButton @click="showModal = false">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="submitLoading" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped></style>
