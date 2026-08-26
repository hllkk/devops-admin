<script setup lang="ts">
import { computed, nextTick, ref } from 'vue';
import { fetchBatchDeleteModel, fetchGetModelList } from '@/service/api/gateway';
import { $t } from '@/locales';
import { MODEL_CATEGORY_OPTIONS, getProviderIcon } from '@/constants/business/gateway';
import ButtonIcon from '@/components/custom/button-icon.vue';
import SvgIcon from '@/components/custom/svg-icon.vue';
import TableSiderLayout from '@/components/advanced/table-sider-layout.vue';
import ModelOperateDrawer from './modules/model-operate-drawer.vue';
import ModelDetailPanel from './modules/model-detail-panel.vue';
import RouterSettingsDialog from './modules/router-settings-dialog.vue';

defineOptions({ name: 'ModelList' });

const searchParams = ref<Api.Gateway.ModelSearchParams>({
  pageNum: 1,
  pageSize: 100,
  name: null,
  modelKey: null,
  category: null,
  isActive: null,
  isPublished: null,
  params: {}
});

const modelList = ref<Api.Gateway.Model[]>([]);
const modelLoading = ref(false);
const selectedModel = ref<Api.Gateway.Model | null>(null);

async function getModelData() {
  modelLoading.value = true;
  const { data, error } = await fetchGetModelList(searchParams.value);
  if (!error && data) {
    modelList.value = data.rows;
    if (selectedModel.value) {
      selectedModel.value = modelList.value.find(m => m.modelId === selectedModel.value!.modelId) ?? null;
    }
  }
  modelLoading.value = false;
}

/** 提交(新增/编辑)后选中目标模型并滚动到可见位置 */
async function selectAndScrollToModel(modelId: CommonType.IdType | null) {
  if (!modelId) return;
  const row = modelList.value.find(m => m.modelId === modelId);
  if (!row) return;
  selectedModel.value = row;
  await nextTick();
  document.querySelector(`[data-model-id="${modelId}"]`)?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
}

getModelData();

/** 左侧列表按类别分组：已知类别按选项顺序，未知类别兜底「其他」 */
const groupedModels = computed(() => {
  const known = MODEL_CATEGORY_OPTIONS.map(opt => ({
    label: $t(opt.label),
    models: modelList.value.filter(m => m.category === opt.value)
  }));
  const knownValues = new Set<string>(MODEL_CATEGORY_OPTIONS.map(o => o.value));
  const others = modelList.value.filter(m => !knownValues.has(m.category));
  if (others.length) known.push({ label: $t('page.gateway.common.categoryOther'), models: others });
  return known.filter(g => g.models.length);
});

// 模型增/改/删
const drawerVisible = ref(false);
const operateType = ref<NaiveUI.TableOperateType>('add');
const editingData = ref<Api.Gateway.Model | null>(null);

function handleAdd() {
  operateType.value = 'add';
  editingData.value = null;
  drawerVisible.value = true;
}
function handleEdit(row: Api.Gateway.Model) {
  operateType.value = 'edit';
  editingData.value = row;
  drawerVisible.value = true;
}
async function handleDelete(row: Api.Gateway.Model) {
  const { error } = await fetchBatchDeleteModel([row.modelId!]);
  if (error) return;
  if (selectedModel.value?.modelId === row.modelId) selectedModel.value = null;
  getModelData();
}
async function handleSubmitted(modelId: CommonType.IdType | null) {
  drawerVisible.value = false;
  await getModelData();
  await selectAndScrollToModel(modelId);
}

function handleDeploymentChanged() {
  getModelData();
}

// 路由策略弹窗(全局配置)
const routerSettingsVisible = ref(false);
function openRouterSettings() {
  routerSettingsVisible.value = true;
}
</script>

<template>
  <TableSiderLayout :sider-title="$t('page.gateway.model.title')">
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
        @click.stop="getModelData"
      />
      <ButtonIcon
        size="small"
        icon="material-symbols:alt-route-outline"
        class="h-28px text-icon"
        :tooltip-content="$t('page.gateway.router.title')"
        @click.stop="openRouterSettings"
      />
    </template>
    <template #sider>
      <NInput
        v-model:value="searchParams.name"
        clearable
        :placeholder="$t('common.keywordSearch')"
        class="mb-8px"
        @update:value="getModelData"
      />
      <NSpin :show="modelLoading" size="small">
        <div class="flex flex-col gap-4px overflow-y-auto" style="max-height: calc(100vh - 300px)">
          <template v-for="group in groupedModels" :key="group.label">
            <div class="sticky top-0 z-1 bg-white py-4px text-11px font-500 text-slate-400 dark:bg-[#18181c]">
              {{ group.label }} · {{ group.models.length }}
            </div>
            <div
              v-for="row in group.models"
              :key="row.modelId"
              class="model-item"
              :class="{ 'is-selected': selectedModel?.modelId === row.modelId }"
              :data-model-id="row.modelId"
              @click="selectedModel = row"
            >
              <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-100 dark:bg-slate-800">
                <SvgIcon :local-icon="getProviderIcon(row.logoProviderType)" class="h-24px w-24px" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-6px">
                  <span class="truncate text-13px font-500">{{ row.name }}</span>
                  <span v-if="!row.isActive" class="shrink-0 rounded bg-slate-100 px-4px text-10px text-slate-400 dark:bg-slate-800">
                    {{ $t('page.gateway.common.inactive') }}
                  </span>
                  <span
                    v-else-if="!row.isPublished"
                    class="shrink-0 rounded bg-slate-100 px-4px text-10px text-slate-400 dark:bg-slate-800"
                  >
                    {{ $t('page.gateway.common.unpublished') }}
                  </span>
                </div>
                <span class="text-11px text-slate-400">
                  {{ row.modelKey || $t('page.gateway.model.modelKeyUnset') }} · {{ row.activeDeploymentCount ?? 0 }}/{{ row.deploymentCount ?? 0 }}
                  {{ $t('page.gateway.deployment.manageTitle') }}
                </span>
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
          </template>
          <NEmpty v-if="!modelLoading && !modelList.length" :description="$t('common.noData')" class="py-24px" />
        </div>
      </NSpin>
    </template>
    <div class="h-full flex-col-stretch overflow-hidden">
      <ModelDetailPanel v-if="selectedModel" :model="selectedModel" @changed="handleDeploymentChanged" />
      <NCard v-else :bordered="false" size="small" class="card-wrapper h-full" content-style="height: 100%">
        <div class="h-full flex-center">
          <NEmpty :description="$t('page.gateway.model.selectModelTip')" />
        </div>
      </NCard>
    </div>
    <ModelOperateDrawer
      v-model:visible="drawerVisible"
      :operate-type="operateType"
      :row-data="editingData"
      @submitted="handleSubmitted"
    />
    <RouterSettingsDialog v-model:visible="routerSettingsVisible" />
  </TableSiderLayout>
</template>

<style scoped>
.model-item {
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

.model-item:hover {
  background-color: rgb(var(--primary-color) / 0.05);
}

.model-item.is-selected {
  border-color: rgb(var(--primary-color) / 0.55);
  background-color: rgb(var(--primary-color) / 0.08);
}
</style>
