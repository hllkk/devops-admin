<script setup lang="ts">
import { computed, ref } from 'vue';
import { fetchGetCostOverview } from '@/service/api/gateway';
import CostDetailTable from './modules/cost-detail-table.vue';
import CostOverview from './modules/cost-overview.vue';
import CostSearch from './modules/cost-search.vue';
import CostTopUsers from './modules/cost-top-users.vue';

defineOptions({ name: 'GatewayCost' });

/** 筛选条件(时间业务日;其余为可选维度过滤),经 cost-search 维护 */
const searchParams = ref<Api.Gateway.CostSearchParams>({
  startDate: null,
  endDate: null,
  departmentId: null,
  userId: null,
  aiKeyId: null,
  model: null,
  provider: null
});

const overview = ref<Api.Gateway.CostOverview>();
const overviewLoading = ref(false);
const detailRef = ref<InstanceType<typeof CostDetailTable>>();
const topRef = ref<InstanceType<typeof CostTopUsers>>();

/** 提交给后端的筛选(空值剔除;分页由明细表自管) */
const filters = computed<Api.Gateway.CostSearchParams>(() => {
  const { startDate, endDate, departmentId, userId, aiKeyId, model, provider } = searchParams.value;
  const params: Api.Gateway.CostSearchParams = {};
  if (startDate) params.startDate = startDate;
  if (endDate) params.endDate = endDate;
  if (departmentId) params.departmentId = departmentId;
  if (userId) params.userId = userId;
  if (aiKeyId) params.aiKeyId = aiKeyId;
  if (model) params.model = model;
  if (provider) params.provider = provider;
  return params;
});

async function loadOverview() {
  overviewLoading.value = true;
  const { data, error } = await fetchGetCostOverview(filters.value);
  overviewLoading.value = false;
  if (error || !data) return;
  overview.value = data;
}

/** 筛选变更:总览/明细/Top 并行刷新 */
function handleSearch() {
  loadOverview();
  detailRef.value?.refresh();
  topRef.value?.refresh();
}

loadOverview();
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <CostSearch v-model:model="searchParams" @search="handleSearch" />
    <NSpin :show="overviewLoading">
      <CostOverview :data="overview" />
    </NSpin>
    <CostTopUsers ref="topRef" :filters="filters" />
    <CostDetailTable ref="detailRef" :filters="filters" />
  </div>
</template>

<style scoped></style>
