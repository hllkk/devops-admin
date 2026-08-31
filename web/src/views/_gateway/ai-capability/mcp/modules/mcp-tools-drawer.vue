<script setup lang="tsx">
import { computed, ref, watch } from 'vue';
import {
  fetchGetMCPServer,
  fetchRefreshMCPTools,
  fetchUpdateMCPToolBilling
} from '@/service/api/gateway';
import { $t } from '@/locales';
import { MCP_BILLING_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'MCPToolsDrawer' });

interface Props {
  row: Api.Gateway.MCPServer | null;
}

const props = defineProps<Props>();

interface Emits {
  (e: 'changed'): void;
}

const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', {
  default: false
});

const loading = ref(false);
const tools = ref<Api.Gateway.MCPTool[]>([]);

const billingOptions = computed(() => [
  { label: $t('page.gateway.mcp.toolsDrawer.inheritServer'), value: '' },
  ...MCP_BILLING_OPTIONS.map(o => ({ label: $t(o.label), value: o.value }))
]);

async function loadTools() {
  if (!props.row?.mcpServerId) return;
  loading.value = true;
  const { data, error } = await fetchGetMCPServer(props.row.mcpServerId);
  if (!error && data) {
    tools.value = data.tools ?? [];
  }
  loading.value = false;
}

watch(visible, val => {
  if (val) loadTools();
});

/** 远端全量刷新(按 tool_name 保留计费配置) */
const refreshing = ref(false);

async function handleRefresh() {
  if (!props.row?.mcpServerId) return;
  refreshing.value = true;
  const { data, error } = await fetchRefreshMCPTools(props.row.mcpServerId);
  refreshing.value = false;
  if (error) return;
  if (data) tools.value = data;
  window.$message?.success($t('page.gateway.mcp.toolsDrawer.refreshSuccess', { count: data?.length ?? 0 }));
  emit('changed');
}

// 计费编辑弹窗(编辑中的工具 → 表单值)
const billingEditVisible = ref(false);
const editingTool = ref<Api.Gateway.MCPTool | null>(null);
const editBillingType = ref('');
const editCost = ref<number | null>(null);
const editInternalCost = ref<number | null>(null);

function openBillingEdit(tool: Api.Gateway.MCPTool) {
  editingTool.value = tool;
  editBillingType.value = tool.billingType ?? '';
  editCost.value = tool.externalCostPerCall;
  editInternalCost.value = tool.internalCostPerCall;
  billingEditVisible.value = true;
}

async function saveBillingEdit() {
  if (!editingTool.value?.mcpToolId) return;
  const isPerCall = editBillingType.value === 'per_call';
  const cost = isPerCall ? editCost.value : null;
  if (isPerCall && (cost === null || cost === undefined)) {
    window.$message?.warning($t('page.gateway.mcp.form.costRequired'));
    return;
  }
  const { error } = await fetchUpdateMCPToolBilling(editingTool.value.mcpToolId, {
    billingType: editBillingType.value,
    externalCostPerCall: cost,
    internalCostPerCall: isPerCall ? editInternalCost.value : null
  });
  if (error) return;
  window.$message?.success($t('common.updateSuccess'));
  billingEditVisible.value = false;
  loadTools();
}

function toolRowKey(row: Api.Gateway.MCPTool) {
  return row.mcpToolId;
}

/** 计费列展示(空=继承服务器) */
function billingLabel(tool: Api.Gateway.MCPTool): string {
  if (!tool.billingType) return $t('page.gateway.mcp.toolsDrawer.inheritServer');
  if (tool.billingType === 'free') return $t('page.gateway.mcp.billingFree');
  return $t('page.gateway.mcp.billingPerCall');
}

const columns = computed(() => [
  {
    type: 'expand' as const,
    renderExpand: (row: Api.Gateway.MCPTool) => {
      if (!row.inputSchema || Object.keys(row.inputSchema).length === 0) {
        return <div class="px-16px py-8px text-gray-400">{$t('page.gateway.mcp.toolsDrawer.schemaEmpty')}</div>;
      }
      return (
        <div class="px-16px py-8px">
          <div class="mb-4px">{$t('page.gateway.mcp.toolsDrawer.schema')}：</div>
          <pre class="max-h-240px overflow-auto break-all whitespace-pre-wrap text-12px">
            {JSON.stringify(row.inputSchema, null, 2)}
          </pre>
        </div>
      );
    }
  },
  {
    key: 'toolName',
    title: $t('page.gateway.mcp.toolsDrawer.col.toolName'),
    minWidth: 170,
    render: (row: Api.Gateway.MCPTool) => <span class="font-mono text-12px">{row.toolName}</span>
  },
  {
    key: 'displayName',
    title: $t('page.gateway.mcp.toolsDrawer.col.displayName'),
    minWidth: 120,
    render: (row: Api.Gateway.MCPTool) => row.displayName || row.toolName
  },
  {
    key: 'description',
    title: $t('page.gateway.mcp.toolsDrawer.col.description'),
    minWidth: 200,
    ellipsis: { tooltip: true }
  },
  {
    key: 'billingType',
    title: $t('page.gateway.mcp.toolsDrawer.col.billingType'),
    width: 110,
    align: 'center' as const,
    render: (row: Api.Gateway.MCPTool) => billingLabel(row)
  },
  {
    key: 'externalCostPerCall',
    title: $t('page.gateway.mcp.toolsDrawer.col.cost'),
    width: 110,
    align: 'right' as const,
    render: (row: Api.Gateway.MCPTool) => (row.externalCostPerCall != null ? `¥${row.externalCostPerCall}` : '-')
  },
  {
    key: 'actions',
    title: $t('common.action'),
    width: 90,
    align: 'center' as const,
    render: (row: Api.Gateway.MCPTool) => (
      <button type="button" class="text-12px color-primary hover:opacity-80" onClick={() => openBillingEdit(row)}>
        {$t('page.gateway.mcp.toolsDrawer.editBilling')}
      </button>
    )
  }
]);
</script>

<template>
  <NDrawer v-model:show="visible" display-directive="show" :width="760" class="max-w-95%">
    <NDrawerContent :title="`${$t('page.gateway.mcp.toolsDrawer.title')} · ${row?.name ?? ''}`" :native-scrollbar="false" closable>
      <div class="mb-12px flex items-center justify-between">
        <span class="text-12px text-slate-400">{{ $t('page.gateway.mcp.toolsDrawer.tip') }}</span>
        <NButton size="small" type="primary" ghost :loading="refreshing" @click="handleRefresh">
          {{ $t('page.gateway.mcp.toolsDrawer.refresh') }}
        </NButton>
      </div>
      <NDataTable
        :data="tools"
        :columns="columns"
        :loading="loading"
        size="small"
        :row-key="toolRowKey"
      >
        <NEmpty v-if="!loading && !tools.length" :description="$t('page.gateway.mcp.toolsDrawer.emptyTip')" class="py-32px" />
      </NDataTable>

      <NModal v-model:show="billingEditVisible" preset="card" :title="$t('page.gateway.mcp.toolsDrawer.editBilling')" class="w-420px max-w-90%">
        <NForm label-placement="left" :label-width="90">
          <NFormItem :label="$t('page.gateway.mcp.col.billingType')">
            <NSelect v-model:value="editBillingType" :options="billingOptions" />
          </NFormItem>
          <NFormItem v-if="editBillingType === 'per_call'" :label="$t('page.gateway.mcp.costPerCall')">
            <NInputNumber v-model:value="editCost" :min="0" :precision="6" class="w-full" />
          </NFormItem>
          <NFormItem v-if="editBillingType === 'per_call'" :label="$t('page.gateway.mcp.internalCostPerCall')">
            <NInputNumber v-model:value="editInternalCost" :min="0" :precision="6" class="w-full" />
          </NFormItem>
        </NForm>
        <template #footer>
          <NSpace justify="end" :size="12">
            <NButton size="small" @click="billingEditVisible = false">{{ $t('common.cancel') }}</NButton>
            <NButton size="small" type="primary" @click="saveBillingEdit">{{ $t('common.confirm') }}</NButton>
          </NSpace>
        </template>
      </NModal>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped></style>
