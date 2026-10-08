<script setup lang="tsx">
import { computed, onMounted, ref } from 'vue';
import { NTag, NTime } from 'naive-ui';
import {
  fetchBatchDeleteAssets,
  fetchCreateAsset,
  fetchGetAssetList,
  fetchGetCredentialOptions,
  fetchUpdateAsset
} from '@/service/api/server';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import {
  ACTIVE_OPTIONS,
  AGENT_STATUS_OPTIONS,
  ASSET_ENV_OPTIONS,
  ASSET_OS_OPTIONS,
  ASSET_TYPE_OPTIONS,
  MONITOR_STATUS_OPTIONS
} from '@/constants/business/server';
import ButtonIcon from '@/components/custom/button-icon.vue';

defineOptions({ name: 'ServerAssetListPanel' });

const appStore = useAppStore();

const searchParams = ref<Api.Server.AssetSearchParams>({
  pageNum: 1,
  pageSize: 10,
  assetName: null,
  manageIp: null,
  assetType: null,
  env: null,
  monitorStatus: null,
  isActive: null,
  params: {}
});

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => fetchGetAssetList(searchParams.value),
  transform: response => defaultTransform(response),
  onPaginationParamsChange: params => {
    searchParams.value.pageNum = params.page;
    searchParams.value.pageSize = params.pageSize;
  },
  columns: () => [
    {
      key: 'assetName',
      title: $t('page.server.asset.col.assetName'),
      align: 'left',
      minWidth: 180,
      render: row => (
        <div class="flex flex-col">
          <span class="font-500">{row.assetName}</span>
          <span class="text-12px text-slate-400">{row.manageIp || '-'}</span>
        </div>
      )
    },
    {
      key: 'assetType',
      title: $t('page.server.asset.col.assetType'),
      align: 'center',
      minWidth: 100,
      render: row => <NTag size="small">{assetTypeLabel(row.assetType)}</NTag>
    },
    {
      key: 'env',
      title: $t('page.server.asset.col.env'),
      align: 'center',
      minWidth: 80,
      render: row => (row.env ? <NTag size="small" type="info" bordered={false}>{row.env}</NTag> : <span class="text-slate-400">-</span>)
    },
    {
      key: 'location',
      title: $t('page.server.asset.col.location'),
      align: 'center',
      minWidth: 120,
      ellipsis: { tooltip: true },
      render: row => row.location || <span class="text-slate-400">-</span>
    },
    {
      key: 'monitorStatus',
      title: $t('page.server.asset.col.monitorStatus'),
      align: 'center',
      minWidth: 90,
      render: row => {
        const type = row.monitorStatus === 'online' ? 'success' : row.monitorStatus === 'offline' ? 'error' : 'default';
        return <NTag size="small" type={type}>{monitorStatusLabel(row.monitorStatus)}</NTag>;
      }
    },
    {
      key: 'agentStatus',
      title: $t('page.server.asset.col.agentStatus'),
      align: 'center',
      minWidth: 90,
      render: row => {
        const type = row.agentStatus === 'running' ? 'success' : row.agentStatus === 'lost' ? 'error' : row.agentStatus === 'installing' ? 'warning' : 'default';
        return <NTag size="small" type={type}>{agentStatusLabel(row.agentStatus)}</NTag>;
      }
    },
    {
      key: 'isActive',
      title: $t('page.server.asset.col.isActive'),
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
            onPositiveClick={() => handleDelete(row.assetId!)}
          />
        </div>
      )
    }
  ]
});

const { checkedRowKeys, onBatchDeleted, onDeleted } = useTableOperate(data, 'assetId', getData);

// ── 状态/类型选项 label ──
const assetTypeOptions = computed(() => ASSET_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const monitorStatusOptions = computed(() => MONITOR_STATUS_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const envOptions = computed(() => ASSET_ENV_OPTIONS.map(o => ({ label: o.label, value: o.value })));
const osOptions = computed(() => ASSET_OS_OPTIONS.map(o => ({ label: o.label, value: o.value })));
const activeOptions = computed(() => ACTIVE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

function assetTypeLabel(v: string): string {
  return assetTypeOptions.value.find(o => o.value === v)?.label ?? v;
}
function monitorStatusLabel(v: string): string {
  return monitorStatusOptions.value.find(o => o.value === v)?.label ?? v;
}
const agentStatusLabels = computed(() => {
  const m: Record<string, string> = {};
  for (const o of AGENT_STATUS_OPTIONS) m[o.value] = $t(o.label);
  return m;
});
function agentStatusLabel(v: string): string {
  return agentStatusLabels.value[v] ?? v;
}

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
    assetName: null,
    manageIp: null,
    assetType: null,
    env: null,
    monitorStatus: null,
    isActive: null,
    params: {}
  };
  getDataByPage();
}

// ── 新增/编辑弹窗 ──
type AssetModel = Api.Server.AssetOperateParams;

const showModal = ref(false);
const submitLoading = ref(false);
const editingAsset = ref<Api.Server.Asset | null>(null);
const assetModel = ref<AssetModel>(createDefaultAssetModel());

const { formRef, validate, restoreValidation } = useNaiveForm();
const { createRequiredRule } = useFormRules();

const rules: Record<'assetName' | 'assetType' | 'manageIp', App.Global.FormRule> = {
  assetName: createRequiredRule($t('page.server.asset.form.assetNameRequired')),
  assetType: createRequiredRule($t('page.server.asset.form.assetTypeRequired')),
  manageIp: createRequiredRule($t('page.server.asset.form.manageIpRequired'))
};

const modalTitle = computed(() => (editingAsset.value ? $t('page.server.asset.edit') : $t('page.server.asset.add')));

// SSH 凭据下拉(仅启用中,异步加载)
const credentialOptions = ref<{ label: string; value: string }[]>([]);

async function loadCredentialOptions() {
  const { error, data: opts } = await fetchGetCredentialOptions('ssh');
  if (!error && opts) {
    credentialOptions.value = opts.map(o => ({ label: o.credentialName, value: String(o.credentialId) }));
  }
}

function createDefaultAssetModel(): AssetModel {
  return {
    assetId: null,
    assetName: '',
    assetType: null,
    manageIp: '',
    sshPort: 22,
    osType: 'linux',
    env: null,
    location: '',
    isActive: true,
    credentialId: null,
    description: ''
  };
}

function handleAdd() {
  editingAsset.value = null;
  assetModel.value = createDefaultAssetModel();
  showModal.value = true;
  restoreValidation();
}

function handleEdit(row: Api.Server.Asset) {
  editingAsset.value = row;
  assetModel.value = {
    assetId: row.assetId,
    assetName: row.assetName,
    assetType: row.assetType,
    manageIp: row.manageIp,
    sshPort: row.sshPort || 22,
    osType: row.osType || 'linux',
    env: row.env || null,
    location: row.location,
    isActive: row.isActive,
    credentialId: row.credentialId && row.credentialId !== '0' ? String(row.credentialId) : null,
    description: row.description
  };
  showModal.value = true;
  restoreValidation();
}

async function handleSubmit() {
  await validate();
  submitLoading.value = true;
  const isEdit = !!editingAsset.value;
  const payload: Api.Server.AssetOperateParams = {
    ...assetModel.value,
    credentialId: assetModel.value.credentialId || '0'
  };
  const { error } = isEdit ? await fetchUpdateAsset(payload) : await fetchCreateAsset(payload);
  submitLoading.value = false;
  if (error) return;
  window.$message?.success($t(isEdit ? 'common.updateSuccess' : 'common.addSuccess'));
  showModal.value = false;
  getData();
  notifyChanged();
}

async function handleDelete(assetId: CommonType.IdType) {
  const { error } = await fetchBatchDeleteAssets([assetId]);
  if (error) return;
  onDeleted();
}

async function handleBatchDelete() {
  const { error } = await fetchBatchDeleteAssets(checkedRowKeys.value);
  if (error) return;
  onBatchDeleted();
}

// 资产增/删后凭据面板的关联计数可能变化——由父组件监听 changed 事件处理
const emit = defineEmits<{
  (e: 'changed'): void;
}>();

function notifyChanged() {
  emit('changed');
}

defineExpose({ refresh: getData });

onMounted(() => {
  getData();
  loadCredentialOptions();
});
</script>

<template>
  <div class="h-full flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <NCard :bordered="false" size="small" class="card-wrapper">
      <NCollapse>
        <NCollapseItem :title="$t('common.search')" name="asset-search">
          <NForm label-placement="left" :label-width="80">
            <NGrid responsive="screen" item-responsive>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.server.asset.col.assetName')" class="pr-24px">
                <NInput v-model:value="searchParams.assetName" clearable :placeholder="$t('common.keywordSearch')" @keyup.enter="() => getDataByPage()" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.server.asset.col.manageIp')" class="pr-24px">
                <NInput v-model:value="searchParams.manageIp" clearable :placeholder="$t('page.server.asset.form.manageIpPlaceholder')" @keyup.enter="() => getDataByPage()" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.server.asset.col.assetType')" class="pr-24px">
                <NSelect v-model:value="searchParams.assetType" clearable :options="assetTypeOptions" :placeholder="$t('common.placeholderSelect')" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.server.asset.col.env')" class="pr-24px">
                <NSelect v-model:value="searchParams.env" clearable :options="envOptions" :placeholder="$t('common.placeholderSelect')" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.server.asset.col.monitorStatus')" class="pr-24px">
                <NSelect v-model:value="searchParams.monitorStatus" clearable :options="monitorStatusOptions" :placeholder="$t('common.placeholderSelect')" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:8" :label="$t('page.server.asset.col.isActive')" class="pr-24px">
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
    <NCard :title="$t('page.server.asset.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
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
        :row-key="row => row.assetId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
    </NCard>
    <NModal v-model:show="showModal" preset="card" :title="modalTitle" class="w-640px">
      <NForm ref="formRef" :model="assetModel" :rules="rules" label-placement="left" :label-width="90">
        <NGrid responsive="screen" item-responsive :x-gap="12">
          <NFormItemGi span="24 m:12" :label="$t('page.server.asset.col.assetName')" path="assetName">
            <NInput v-model:value="assetModel.assetName" :placeholder="$t('page.server.asset.form.assetNamePlaceholder')" />
          </NFormItemGi>
          <NFormItemGi span="24 m:12" :label="$t('page.server.asset.col.assetType')" path="assetType">
            <NSelect v-model:value="assetModel.assetType" :options="assetTypeOptions" :placeholder="$t('common.placeholderSelect')" />
          </NFormItemGi>
          <NFormItemGi span="24 m:12" :label="$t('page.server.asset.col.manageIp')" path="manageIp">
            <NInput v-model:value="assetModel.manageIp" :placeholder="$t('page.server.asset.form.manageIpPlaceholder')" />
          </NFormItemGi>
          <NFormItemGi span="24 m:12" :label="$t('page.server.asset.col.sshPort')" path="sshPort">
            <NInputNumber v-model:value="assetModel.sshPort" :min="1" :max="65535" class="w-full" :placeholder="$t('page.server.asset.form.sshPortPlaceholder')" />
          </NFormItemGi>
          <NFormItemGi span="24 m:12" :label="$t('page.server.asset.col.osType')" path="osType">
            <NSelect v-model:value="assetModel.osType" :options="osOptions" :placeholder="$t('common.placeholderSelect')" />
          </NFormItemGi>
          <NFormItemGi span="24 m:12" :label="$t('page.server.asset.col.env')" path="env">
            <NSelect v-model:value="assetModel.env" clearable tag filterable :options="envOptions" :placeholder="$t('page.server.asset.form.envPlaceholder')" />
          </NFormItemGi>
          <NFormItemGi span="24 m:12" :label="$t('page.server.asset.col.location')" path="location">
            <NInput v-model:value="assetModel.location" :placeholder="$t('page.server.asset.form.locationPlaceholder')" />
          </NFormItemGi>
          <NFormItemGi span="24 m:12" :label="$t('page.server.asset.col.credential')" path="credentialId">
            <NSelect
              v-model:value="assetModel.credentialId"
              clearable
              :options="credentialOptions"
              :placeholder="$t('page.server.asset.form.credentialPlaceholder')"
            />
          </NFormItemGi>
          <NFormItemGi span="24" :label="$t('page.server.asset.col.description')" path="description">
            <NInput v-model:value="assetModel.description" type="textarea" :rows="2" :placeholder="$t('page.server.asset.form.descPlaceholder')" />
          </NFormItemGi>
          <NFormItemGi span="24" :label="$t('page.server.asset.col.isActive')" path="isActive">
            <NSwitch :value="!!assetModel.isActive" @update:value="v => (assetModel.isActive = v)" />
          </NFormItemGi>
        </NGrid>
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
