<script setup lang="ts">
import { ref } from 'vue';
import { fetchBatchDeleteProvider, fetchGetProviderList, fetchUpdateProvider } from '@/service/api/gateway';
import { $t } from '@/locales';
import { BALANCE_SYNC_PROVIDER_TYPES, getProviderIcon } from '@/constants/business/gateway';
import ButtonIcon from '@/components/custom/button-icon.vue';
import SvgIcon from '@/components/custom/svg-icon.vue';
import TableSiderLayout from '@/components/advanced/table-sider-layout.vue';
import ProviderOperateDrawer from './modules/provider-operate-drawer.vue';
import CredentialPanel from './modules/credential-panel.vue';
import ProviderBalancePanel from './modules/provider-balance-panel.vue';

defineOptions({ name: 'ProviderList' });

const searchParams = ref<Api.Gateway.ProviderSearchParams>({
  pageNum: 1,
  pageSize: 100,
  name: null,
  providerType: null,
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
        class="h-28px text-icon color-primary"
        :tooltip-content="$t('common.add')"
        @click.stop="handleAdd"
      />
      <ButtonIcon
        size="small"
        icon="material-symbols:refresh-rounded"
        class="h-28px text-icon"
        :tooltip-content="$t('common.refresh')"
        @click.stop="getProviderData"
      />
    </template>
    <template #sider>
      <NInput
        v-model:value="searchParams.name"
        clearable
        :placeholder="$t('common.keywordSearch')"
        class="mb-8px"
        @update:value="getProviderData"
      />
      <NSpin :show="providerLoading" size="small">
        <div class="flex flex-col gap-4px overflow-y-auto" style="max-height: calc(100vh - 300px)">
          <div
            v-for="row in providerList"
            :key="row.providerId"
            class="provider-item"
            :class="{ 'is-selected': selectedProvider?.providerId === row.providerId }"
            @click="selectedProvider = row"
          >
            <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-100 dark:bg-slate-800">
              <SvgIcon :local-icon="getProviderIcon(row.providerType)" class="h-24px w-24px" />
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-6px">
                <span class="truncate text-13px font-500">{{ row.name }}</span>
                <span v-if="!row.isActive" class="shrink-0 rounded bg-slate-100 px-4px text-10px text-slate-400 dark:bg-slate-800">
                  {{ $t('page.gateway.common.inactive') }}
                </span>
              </div>
              <span class="text-11px text-slate-400">{{ row.providerType }} · {{ row.credentialCount }} {{ $t('page.gateway.credential.title') }}</span>
            </div>
            <div class="flex-center gap-12px" @click.stop>
              <ButtonIcon
                text
                type="primary"
                size="small"
                icon="material-symbols:drive-file-rename-outline-outline"
                :tooltip-content="$t('common.edit')"
                @click="handleEdit(row)"
              />
              <ButtonIcon
                text
                type="error"
                size="small"
                icon="material-symbols:delete-outline"
                :tooltip-content="$t('common.delete')"
                :popconfirm-content="$t('common.confirmDelete')"
                @positive-click="handleDelete(row)"
              />
            </div>
          </div>
          <NEmpty v-if="!providerLoading && !providerList.length" :description="$t('common.noData')" class="py-24px" />
        </div>
      </NSpin>
    </template>
    <div class="h-full flex flex-col gap-12px overflow-hidden">
      <template v-if="selectedProvider">
        <CredentialPanel
          class="min-h-0 flex-1"
          :provider="selectedProvider"
          @toggle-provider="handleToggleProvider(selectedProvider!)"
          @changed="handleCredentialChanged"
        />
        <!-- 套餐余量旁路(仅支持采集的供应商类型展示,厂商侧口径与网关标价成本互不并) -->
        <ProviderBalancePanel
          v-if="BALANCE_SYNC_PROVIDER_TYPES.has(selectedProvider.providerType)"
          class="flex-shrink-0"
          :provider="selectedProvider"
        />
      </template>
      <NCard v-else :bordered="false" size="small" class="card-wrapper min-h-0 flex-1" content-style="height: 100%">
        <div class="h-full flex-center">
          <NEmpty :description="$t('page.gateway.provider.selectProviderTip')" />
        </div>
      </NCard>
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
.provider-item {
  display: flex;
  cursor: pointer;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border: 1px solid transparent;
  border-radius: 8px;
  transition:
    background-color 0.2s,
    border-color 0.2s;
}

.provider-item:hover {
  background-color: rgb(var(--primary-color) / 0.05);
}

.provider-item.is-selected {
  border-color: rgb(var(--primary-color) / 0.55);
  background-color: rgb(var(--primary-color) / 0.08);
}
</style>
