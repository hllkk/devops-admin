<script setup lang="tsx">
import { computed, ref } from 'vue';
import { jsonClone } from '@sa/utils';
import { NTag, NTime } from 'naive-ui';
import {
  fetchBatchDeleteKeyScenario,
  fetchCreateKeyScenario,
  fetchGetKeyScenarioList,
  fetchUpdateKeyScenario
} from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import ButtonIcon from '@/components/custom/button-icon.vue';
import { ACTIVE_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'KeyScenarioPanel' });

const appStore = useAppStore();

const searchParams = ref<Api.Gateway.KeyScenarioSearchParams>({
  pageNum: 1,
  pageSize: 10,
  name: null,
  isActive: null,
  params: {}
});

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => fetchGetKeyScenarioList(searchParams.value),
  transform: response => defaultTransform(response),
  onPaginationParamsChange: params => {
    searchParams.value.pageNum = params.page;
    searchParams.value.pageSize = params.pageSize;
  },
  columns: () => [
    {
      type: 'selection',
      align: 'center',
      width: 48
    },
    {
      key: 'index',
      title: $t('common.index'),
      align: 'center',
      width: 64,
      render: (_, index) => index + 1
    },
    {
      key: 'name',
      title: $t('page.gateway.keyScenario.col.name'),
      align: 'center',
      minWidth: 140,
      ellipsis: { tooltip: true }
    },
    {
      key: 'description',
      title: $t('page.gateway.keyScenario.col.description'),
      align: 'center',
      minWidth: 220,
      ellipsis: { tooltip: true },
      render: row => row.description || <span class="text-slate-400">-</span>
    },
    {
      key: 'isActive',
      title: $t('page.gateway.aiKey.col.isActive'),
      align: 'center',
      minWidth: 90,
      render: row => <NTag type={row.isActive ? 'success' : 'default'}>{$t(row.isActive ? 'page.gateway.common.active' : 'page.gateway.common.inactive')}</NTag>
    },
    {
      key: 'createTime',
      title: $t('page.gateway.common.createTime'),
      align: 'center',
      minWidth: 170,
      render: row => <NTime time={Date.parse(row.createTime)} format="yyyy-MM-dd HH:mm:ss" />
    },
    {
      key: 'operate',
      title: $t('common.operate'),
      align: 'center',
      width: 160,
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
            onPositiveClick={() => handleDelete(row.scenarioId!)}
          />
        </div>
      )
    }
  ]
});

const { checkedRowKeys, onBatchDeleted, onDeleted } = useTableOperate(data, 'scenarioId', getData);

const activeOptions = computed(() => ACTIVE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

const isActiveSearch = computed<number | null>({
  get: () => (searchParams.value.isActive === null ? null : searchParams.value.isActive ? 1 : 0),
  set: v => {
    searchParams.value.isActive = v === null ? null : v === 1;
  }
});

// 编辑弹窗(NModal)：场景仅名称/描述/启停三字段
type ScenarioModel = Api.Gateway.KeyScenarioOperateParams;

const showModal = ref(false);
const submitLoading = ref(false);
const editingScenario = ref<Api.Gateway.KeyScenario | null>(null);
const scenarioModel = ref<ScenarioModel>(createDefaultScenarioModel());

function createDefaultScenarioModel(): ScenarioModel {
  return { scenarioId: null, name: '', description: '', isActive: true };
}

const { formRef, validate, restoreValidation } = useNaiveForm();
const { createRequiredRule } = useFormRules();

const rules: Record<'name', App.Global.FormRule> = {
  name: createRequiredRule($t('page.gateway.keyScenario.form.nameRequired'))
};

const modalTitle = computed(() =>
  editingScenario.value ? $t('page.gateway.keyScenario.edit') : $t('page.gateway.keyScenario.add')
);

function handleEdit(row: Api.Gateway.KeyScenario) {
  editingScenario.value = row;
  scenarioModel.value = {
    scenarioId: row.scenarioId,
    name: row.name,
    description: row.description,
    isActive: row.isActive
  };
  showModal.value = true;
  restoreValidation();
}

function handleAddScenario() {
  editingScenario.value = null;
  scenarioModel.value = createDefaultScenarioModel();
  showModal.value = true;
  restoreValidation();
}

async function handleSubmitScenario() {
  await validate();
  submitLoading.value = true;
  const isEdit = !!editingScenario.value;
  const { error } = isEdit ? await fetchUpdateKeyScenario(scenarioModel.value) : await fetchCreateKeyScenario(scenarioModel.value);
  submitLoading.value = false;
  if (error) return;
  window.$message?.success($t(isEdit ? 'common.updateSuccess' : 'common.addSuccess'));
  showModal.value = false;
  getData();
  notifyChanged();
}

async function handleDelete(scenarioId: CommonType.IdType) {
  const { error } = await fetchBatchDeleteKeyScenario([scenarioId]);
  if (error) return;
  onDeleted();
  notifyChanged();
}

async function handleBatchDelete() {
  const { error } = await fetchBatchDeleteKeyScenario(checkedRowKeys.value);
  if (error) return;
  onBatchDeleted();
  notifyChanged();
}

// 场景变更后，密钥列表的场景列需刷新——由父组件监听 changed 事件处理
const emit = defineEmits<{
  (e: 'changed'): void;
}>();

function notifyChanged() {
  emit('changed');
}
</script>

<template>
  <div class="flex-col-stretch gap-16px">
    <NCard :bordered="false" size="small" class="card-wrapper">
      <NCollapse>
        <NCollapseItem :title="$t('common.search')" name="key-scenario-search">
          <NForm label-placement="left" :label-width="80">
            <NGrid responsive="screen" item-responsive>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.keyScenario.col.name')" class="pr-24px">
                <NInput v-model:value="searchParams.name" clearable :placeholder="$t('common.placeholderInput')" @keyup.enter="() => getDataByPage()" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.aiKey.col.isActive')" class="pr-24px">
                <NSelect v-model:value="isActiveSearch" clearable :options="activeOptions" :placeholder="$t('common.placeholderSelect')" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:8" class="pr-24px">
                <NSpace class="w-full" justify="end">
                  <NButton @click="() => getDataByPage()">
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
    <NCard :title="$t('page.gateway.keyScenario.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
      <template #header-extra>
        <TableHeaderOperation
          v-model:columns="columnChecks"
          :disabled-delete="checkedRowKeys.length === 0"
          :loading="loading"
          :show-add="true"
          :show-delete="true"
          @add="handleAddScenario"
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
        :row-key="row => row.scenarioId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
    </NCard>
    <NModal v-model:show="showModal" preset="card" :title="modalTitle" class="w-480px">
      <NForm ref="formRef" :model="scenarioModel" :rules="rules" label-placement="left" :label-width="80">
        <NFormItem :label="$t('page.gateway.keyScenario.col.name')" path="name">
          <NInput v-model:value="scenarioModel.name" :placeholder="$t('page.gateway.keyScenario.form.namePlaceholder')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.keyScenario.col.description')" path="description">
          <NInput v-model:value="scenarioModel.description" type="textarea" :rows="2" :placeholder="$t('page.gateway.keyScenario.form.descPlaceholder')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.aiKey.col.isActive')" path="isActive">
          <NSwitch :value="!!scenarioModel.isActive" @update:value="v => (scenarioModel.isActive = v)" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace class="w-full" justify="end">
          <NButton @click="showModal = false">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="submitLoading" @click="handleSubmitScenario">{{ $t('common.confirm') }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped></style>
