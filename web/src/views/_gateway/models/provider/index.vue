<script setup lang="tsx">
import { ref } from 'vue';
import { fetchBatchDeleteProvider, fetchGetProviderList, fetchUpdateProvider } from '@/service/api/gateway';
import { $t } from '@/locales';
import ButtonIcon from '@/components/custom/button-icon.vue';
import TableSiderLayout from '@/components/advanced/table-sider-layout.vue';
import ProviderOperateDrawer from './modules/provider-operate-drawer.vue';
import CredentialPanel from './modules/credential-panel.vue';

defineOptions({ name: 'ProviderList' });

const searchParams = ref<Api.Gateway.ProviderSearchParams>({
  pageNum: 1,
  pageSize: 100,
  name: null,
  providerType: null,
  billingType: null,
  isActive: null,
  params: {}
});

const providerList = ref<Api.Gateway.Provider[]>([]);
const providerLoading = ref(false);
const selectedProvider = ref<Api.Gateway.Provider | null>(null);

async function getProviderData() {
  providerLoading.value = true;
  const { data, error } = await fetchGetProviderList(searchParams.value);
  if (!error && data) {
    providerList.value = data.rows;
    if (selectedProvider.value) {
      selectedProvider.value = providerList.value.find(p => p.providerId === selectedProvider.value!.providerId) ?? null;
    }
  }
  providerLoading.value = false;
}

getProviderData();

const columns = [
  {
    key: 'name',
    render: (row: Api.Gateway.Provider) => (
      <div class="flex flex-col gap-2px py-2px">
        <div class="flex items-center gap-6px">
          <span class="text-13px font-500">{row.name}</span>
          {!row.isActive ? <span class="rounded bg-slate-100 px-4px text-10px text-slate-400">{$t('page.gateway.common.inactive')}</span> : null}
        </div>
        <span class="text-11px text-slate-400">{row.providerType} · {row.credentialCount} {$t('page.gateway.credential.title')}</span>
      </div>
    )
  },
  {
    key: 'operate',
    width: 68,
    render: (row: Api.Gateway.Provider) => (
      <div class="flex-center gap-4px">
        <ButtonIcon
          text
          type="primary"
          size="small"
          icon="material-symbols:drive-file-rename-outline-outline"
          tooltip-content={$t('common.edit')}
          onClick={(e: Event) => { e.stopPropagation(); handleEdit(row); }}
        />
        <ButtonIcon
          text
          type="error"
          size="small"
          icon="material-symbols:delete-outline"
          tooltip-content={$t('common.delete')}
          popconfirm-content={$t('common.confirmDelete')}
          onClick={(e: Event) => e.stopPropagation()}
          onPositiveClick={() => handleDelete(row)}
        />
      </div>
    )
  }
];

function rowProps(row: Api.Gateway.Provider) {
  return {
    class: selectedProvider.value?.providerId === row.providerId ? 'n-data-table-tr--selected' : '',
    onClick: () => {
      selectedProvider.value = row;
    }
  };
}

// 供应商增/改/删
const drawerVisible = ref(false);
const operateType = ref<NaiveUI.TableOperateType>('add');
const editingData = ref<Api.Gateway.Provider | null>(null);

function handleAdd() {
  operateType.value = 'add';
  editingData.value = null;
  drawerVisible.value = true;
}
function handleEdit(row: Api.Gateway.Provider) {
  operateType.value = 'edit';
  editingData.value = row;
  drawerVisible.value = true;
}
async function handleDelete(row: Api.Gateway.Provider) {
  const { error } = await fetchBatchDeleteProvider([row.providerId!]);
  if (error) return;
  if (selectedProvider.value?.providerId === row.providerId) selectedProvider.value = null;
  getProviderData();
}
async function handleToggleProvider(row: Api.Gateway.Provider) {
  const { error } = await fetchUpdateProvider({ ...row, isActive: !row.isActive });
  if (error) return;
  getProviderData();
}
function handleSubmitted() {
  drawerVisible.value = false;
  getProviderData();
}

function handleCredentialChanged() {
  getProviderData();
}
</script>

<template>
  <TableSiderLayout :sider-title="$t('page.gateway.provider.title')">
    <template #header-extra>
      <ButtonIcon
        size="small"
        icon="material-symbols:add-rounded"
        class="h-18px text-icon"
        :tooltip-content="$t('common.add')"
        @click.stop="handleAdd"
      />
      <ButtonIcon
        size="small"
        icon="material-symbols:refresh-rounded"
        class="h-18px text-icon"
        :tooltip-content="$t('common.refresh')"
        @click.stop="getProviderData"
      />
    </template>
    <template #sider>
      <NInput
        v-model:value="searchParams.name"
        clearable
        size="small"
        :placeholder="$t('common.keywordSearch')"
        class="mb-8px"
        @update:value="getProviderData"
      />
      <NDataTable
        :columns="columns"
        :data="providerList"
        :loading="providerLoading"
        size="small"
        :row-key="row => row.providerId"
        :row-props="rowProps"
        :max-height="`calc(100vh - 280px)`"
        class="h-full"
      />
    </template>
    <div class="h-full flex-col-stretch overflow-hidden">
      <CredentialPanel
        v-if="selectedProvider"
        :provider="selectedProvider"
        @toggle-provider="handleToggleProvider(selectedProvider!)"
        @changed="handleCredentialChanged"
      />
      <NEmpty v-else :description="$t('page.gateway.provider.selectProviderTip')" class="h-full flex-center" />
    </div>
    <ProviderOperateDrawer
      v-model:visible="drawerVisible"
      :operate-type="operateType"
      :row-data="editingData"
      @submitted="handleSubmitted"
    />
  </TableSiderLayout>
</template>

<style scoped>
:deep(.n-data-table-tr--selected td) {
  background-color: var(--n-color-hover, rgba(99, 179, 237, 0.12)) !important;
}
</style>
