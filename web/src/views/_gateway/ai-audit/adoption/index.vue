<script setup lang="ts">
import { computed, ref } from 'vue';
import { fetchGetAdoptionOverview } from '@/service/api/gateway';
import AdoptionDeptTable from './modules/adoption-dept-table.vue';
import AdoptionModels from './modules/adoption-models.vue';
import AdoptionOverview from './modules/adoption-overview.vue';
import AdoptionSearch from './modules/adoption-search.vue';

defineOptions({ name: 'GatewayAdoption' });

/** 筛选条件(时间业务日+部门含子树),经 adoption-search 维护 */
const searchParams = ref<Api.Gateway.AdoptionSearchParams>({
  startDate: null,
  endDate: null,
  departmentId: null
});

const overview = ref<Api.Gateway.AdoptionOverview>();
const overviewLoading = ref(false);
const deptRef = ref<InstanceType<typeof AdoptionDeptTable>>();
const modelRef = ref<InstanceType<typeof AdoptionModels>>();

/** 提交给后端的筛选(空值剔除) */
const filters = computed<Api.Gateway.AdoptionSearchParams>(() => {
  const { startDate, endDate, departmentId } = searchParams.value;
  const params: Api.Gateway.AdoptionSearchParams = {};
  if (startDate) params.startDate = startDate;
  if (endDate) params.endDate = endDate;
  if (departmentId) params.departmentId = departmentId;
  return params;
});

async function loadOverview() {
  overviewLoading.value = true;
  const { data, error } = await fetchGetAdoptionOverview(filters.value);
  overviewLoading.value = false;
  if (error || !data) return;
  overview.value = data;
}

/** 筛选变更:总览/部门/模型分布并行刷新 */
function handleSearch() {
  loadOverview();
  deptRef.value?.refresh();
  modelRef.value?.refresh();
}

loadOverview();
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <AdoptionSearch v-model:model="searchParams" @search="handleSearch" />
    <NSpin :show="overviewLoading">
      <AdoptionOverview :data="overview" />
    </NSpin>
    <AdoptionDeptTable ref="deptRef" :filters="filters" />
    <AdoptionModels ref="modelRef" :filters="filters" />
  </div>
</template>

<style scoped></style>
