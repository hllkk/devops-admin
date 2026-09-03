<script setup lang="tsx">
import { onMounted, ref } from 'vue';
import { NTag, NTooltip } from 'naive-ui';
import IconUrl from '@/components/custom/icon-url.vue';
import {
  fetchBatchDeleteMCPServer,
  fetchGetMCPServerList,
  fetchHealthCheckMCPServer
} from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable } from '@/hooks/common/table';
import { $t } from '@/locales';
import McpSearch from './modules/mcp-search.vue';
import MCPOperateDrawer from './modules/mcp-operate-drawer.vue';
import MCPPublishDialog from './modules/mcp-publish-dialog.vue';
import MCPToolsDrawer from './modules/mcp-tools-drawer.vue';

defineOptions({ name: 'GatewayMcp' });

const appStore = useAppStore();

const searchParams = ref<Api.Gateway.MCPServerSearchParams>({
  pageNum: 1,
  pageSize: 20,
  name: null,
  category: null,
  isActive: null,
  isPublished: null,
  healthStatus: null,
  params: {}
});

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } =
  useNaivePaginatedTable({
    api: () => {
      const { pageNum, pageSize, name, category, healthStatus } = searchParams.value;
      const params: Api.Gateway.MCPServerSearchParams = { pageNum, pageSize };
      if (name) params.name = name;
      if (category) params.category = category;
      if (healthStatus) params.healthStatus = healthStatus;
      if (searchParams.value.isActive !== null && searchParams.value.isActive !== undefined) {
        params.isActive = searchParams.value.isActive;
      }
      if (searchParams.value.isPublished !== null && searchParams.value.isPublished !== undefined) {
        params.isPublished = searchParams.value.isPublished;
      }
      return fetchGetMCPServerList(params);
    },
    transform: response => defaultTransform(response),
    onPaginationParamsChange: params => {
      searchParams.value.pageNum = params.page;
      searchParams.value.pageSize = params.pageSize;
    },
    columns: () => [
      {
        key: 'name',
        title: $t('page.gateway.mcp.col.name'),
        align: 'center',
        minWidth: 140,
        render: row => (
          <div class="flex items-center justify-center gap-8px">
            <IconUrl value={row.iconUrl} size={24} />
            <div class="flex flex-col items-start">
              <span class="font-500">{row.name}</span>
              <span class="font-mono text-12px text-slate-400">{row.serverName}</span>
            </div>
          </div>
        )
      },
      {
        key: 'category',
        title: $t('page.gateway.mcp.col.category'),
        align: 'center',
        minWidth: 90,
        render: row => <NTag size="small">{row.category}</NTag>
      },
      {
        key: 'toolCount',
        title: $t('page.gateway.mcp.col.toolCount'),
        align: 'center',
        minWidth: 80
      },
      {
        key: 'callCount',
        title: $t('page.gateway.mcp.col.callCount'),
        align: 'right',
        minWidth: 90,
        render: row => row.callCount?.toLocaleString() ?? '0'
      },
      {
        key: 'billingType',
        title: $t('page.gateway.mcp.col.billingType'),
        align: 'center',
        minWidth: 100,
        render: row =>
          row.billingType === 'per_call' ? (
            <NTag size="small" type="warning">
              {$t('page.gateway.mcp.billingPerCall')}
            </NTag>
          ) : (
            <NTag size="small">{$t('page.gateway.mcp.billingFree')}</NTag>
          )
      },
      {
        key: 'healthStatus',
        title: $t('page.gateway.mcp.col.healthStatus'),
        align: 'center',
        minWidth: 100,
        render: row => {
          const typeMap: Record<string, 'success' | 'warning' | 'default'> = {
            healthy: 'success',
            unhealthy: 'warning',
            unknown: 'default'
          };
          const labelMap: Record<string, string> = {
            healthy: $t('page.gateway.mcp.health.healthy'),
            unhealthy: $t('page.gateway.mcp.health.unhealthy'),
            unknown: $t('page.gateway.mcp.health.unknown')
          };
          return <NTag size="small" type={typeMap[row.healthStatus]}>{labelMap[row.healthStatus]}</NTag>;
        }
      },
      {
        key: 'isPublished',
        title: $t('page.gateway.mcp.col.isPublished'),
        align: 'center',
        minWidth: 90,
        render: row =>
          row.isPublished ? (
            <NTag size="small" type="success">
              {$t('page.gateway.common.published')}
            </NTag>
          ) : (
            <NTag size="small">{$t('page.gateway.common.unpublished')}</NTag>
          )
      },
      {
        key: 'isActive',
        title: $t('page.gateway.mcp.col.isActive'),
        align: 'center',
        minWidth: 80,
        render: row =>
          row.isActive ? (
            <NTag size="small" type="success">
              {$t('page.gateway.common.active')}
            </NTag>
          ) : (
            <NTag size="small" type="error">
              {$t('page.gateway.common.inactive')}
            </NTag>
          )
      },
      {
        key: 'litellmSynced',
        title: $t('page.gateway.mcp.col.litellmSynced'),
        align: 'center',
        minWidth: 90,
        render: row => {
          const tag = (
            <NTag size="small" type={row.litellmSynced ? 'success' : 'warning'}>
              {$t(row.litellmSynced ? 'page.gateway.common.synced' : 'page.gateway.common.unsynced')}
            </NTag>
          );
          // 未同步且有错误详情时 hover 可看原因(对标 AIHelms「未同步」徽标)
          if (!row.litellmSynced && row.litellmSyncError) {
            return (
              <NTooltip>
                {{
                  trigger: () => tag,
                  default: () => row.litellmSyncError
                }}
              </NTooltip>
            );
          }
          return tag;
        }
      },
      {
        key: 'actions',
        title: $t('common.action'),
        align: 'center',
        width: 200,
        fixed: 'right',
        render: row => (
          <div class="flex-center gap-6px">
            <button
              type="button"
              class="text-12px color-primary hover:opacity-80"
              onClick={() => handleEdit(row)}
            >
              {$t('common.edit')}
            </button>
            <button
              type="button"
              class="text-12px color-primary hover:opacity-80"
              onClick={() => handlePublish(row)}
            >
              {$t('page.gateway.mcp.publish.short')}
            </button>
            <button
              type="button"
              class="text-12px color-primary hover:opacity-80"
              onClick={() => handleTools(row)}
            >
              {$t('page.gateway.mcp.toolsDrawer.short')}
            </button>
            <button
              type="button"
              class="text-12px color-primary hover:opacity-80"
              onClick={() => handleHealthCheck(row)}
            >
              {$t('page.gateway.mcp.healthCheck.short')}
            </button>
            <NPopconfirm onPositiveClick={() => handleDelete(row)}>
              {{
                default: () => $t('common.confirmDelete'),
                trigger: () => (
                  <button type="button" class="text-12px color-error hover:opacity-80">
                    {$t('common.delete')}
                  </button>
                )
              }}
            </NPopconfirm>
          </div>
        )
      }
    ]
  });

// 新增/编辑抽屉
const drawerVisible = ref(false);
const operateType = ref<NaiveUI.TableOperateType>('add');
const editingData = ref<Api.Gateway.MCPServer | null>(null);

function handleAdd() {
  operateType.value = 'add';
  editingData.value = null;
  drawerVisible.value = true;
}
function handleEdit(row: Api.Gateway.MCPServer) {
  operateType.value = 'edit';
  editingData.value = row;
  drawerVisible.value = true;
}
async function handleDelete(row: Api.Gateway.MCPServer) {
  const { error } = await fetchBatchDeleteMCPServer([row.mcpServerId!]);
  if (error) return;
  window.$message?.success($t('common.deleteSuccess'));
  getData();
}
function handleSubmitted() {
  drawerVisible.value = false;
  getData();
}

// 发布设置弹窗
const publishVisible = ref(false);
const publishRow = ref<Api.Gateway.MCPServer | null>(null);

function handlePublish(row: Api.Gateway.MCPServer) {
  publishRow.value = row;
  publishVisible.value = true;
}
function handlePublished() {
  publishVisible.value = false;
  getData();
}

// 工具面板抽屉
const toolsVisible = ref(false);
const toolsRow = ref<Api.Gateway.MCPServer | null>(null);

function handleTools(row: Api.Gateway.MCPServer) {
  toolsRow.value = row;
  toolsVisible.value = true;
}
function handleToolsChanged() {
  getData();
}

// 健康检查
async function handleHealthCheck(row: Api.Gateway.MCPServer) {
  const { error, data: healthResult } = await fetchHealthCheckMCPServer(row.mcpServerId!);
  if (error) return;
  if (healthResult?.healthStatus === 'healthy') {
    window.$message?.success($t('page.gateway.mcp.healthCheck.done'));
  } else {
    window.$message?.error(
      `${$t('page.gateway.mcp.healthCheck.failed')}: ${healthResult?.healthCheckError || $t('page.gateway.mcp.health.unknown')}`
    );
  }
  getData();
}

onMounted(() => {
  getData();
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <McpSearch v-model:model="searchParams" @search="getDataByPage" />

    <NCard :title="$t('page.gateway.mcp.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
      <template #header-extra>
        <NSpace size="small">
          <NButton size="small" type="primary" @click="handleAdd">
            {{ $t('page.gateway.mcp.add') }}
          </NButton>
          <TableHeaderOperation
            v-model:columns="columnChecks"
            :loading="loading"
            :show-add="false"
            :show-delete="false"
            :show-refresh="true"
            @refresh="getData"
          />
        </NSpace>
      </template>
      <NDataTable
        :columns="columns"
        :data="data"
        size="small"
        :flex-height="!appStore.isMobile"
        :scroll-x="scrollX"
        :loading="loading"
        remote
        :row-key="row => row.mcpServerId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
    </NCard>

    <MCPOperateDrawer
      v-model:visible="drawerVisible"
      :operate-type="operateType"
      :row-data="editingData"
      @submitted="handleSubmitted"
    />
    <MCPPublishDialog v-model:visible="publishVisible" :row="publishRow" @submitted="handlePublished" />
    <MCPToolsDrawer v-model:visible="toolsVisible" :row="toolsRow" @changed="handleToolsChanged" />
  </div>
</template>

<style scoped></style>
