<script setup lang="tsx">
import { ref } from 'vue';
import { NProgress, NTag, NTime } from 'naive-ui';
import { fetchBatchDeleteAiKey, fetchGetAiKeyList, fetchResyncAiKeys, fetchRevealAiKeyValue, fetchRotateAiKey } from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { $t } from '@/locales';
import { handleCopy } from '@/utils/copy';
import ButtonIcon from '@/components/custom/button-icon.vue';
import { KEY_TYPE_OPTIONS, isMainKeyType } from '@/constants/business/gateway';
import AiKeySearch from './ai-key-search.vue';
import AiKeyOperateDrawer from './ai-key-operate-drawer.vue';
import AiKeyBatchModal from './ai-key-batch-modal.vue';

defineOptions({
  name: 'AiKeyListPanel'
});

const appStore = useAppStore();

const searchParams = ref<Api.Gateway.AiKeySearchParams>({
  pageNum: 1,
  pageSize: 10,
  keyType: null,
  ownerType: null,
  ownerId: null,
  name: null,
  isActive: null,
  params: {}
});

const keyTypeLabelKey = (v: string) => KEY_TYPE_OPTIONS.find(o => o.value === v)?.label ?? 'page.gateway.common.keyPersonalScene';

/** 行瞬态字段：完整明文缓存 + 展开态(data 深层 reactive，改行对象即触发单元格重渲；翻页/刷新自然重置) */
type AiKeyRow = Api.Gateway.AiKey & { fullKeyValue?: string; keyRevealed?: boolean };

/** 确保行内已有完整明文(首次点击经 value/:id 按需拉取并缓存，避免明文随列表批量出网) */
async function ensureFullKey(row: AiKeyRow): Promise<string | null> {
  if (row.fullKeyValue) return row.fullKeyValue;
  const { data, error } = await fetchRevealAiKeyValue(row.aiKeyId!);
  if (error) return null;
  row.fullKeyValue = data.keyValue;
  return data.keyValue;
}

async function toggleKeyReveal(row: AiKeyRow) {
  if (row.keyRevealed) {
    row.keyRevealed = false;
    return;
  }
  const plain = await ensureFullKey(row);
  if (plain == null) return;
  row.keyRevealed = true;
}

async function copyFullKey(row: AiKeyRow) {
  const plain = await ensureFullKey(row);
  if (plain == null) return;
  handleCopy(plain);
}

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => fetchGetAiKeyList(searchParams.value),
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
      key: 'ownerUsername',
      title: $t('page.gateway.aiKey.col.username'),
      align: 'center',
      minWidth: 110,
      // 登录用户名(user 归属联表 sys_users；dept 归属无登录名显示 -)
      render: row => row.ownerUsername || <span class="text-slate-400">-</span>
    },
    {
      key: 'keyType',
      title: $t('page.gateway.aiKey.col.keyType'),
      align: 'center',
      minWidth: 120,
      // 类型+场景同属"这是什么 Key"：场景 Key 在类型标签下以小字带出场景名(吸收原独立场景列)
      render: row => {
        const tag = <NTag type={row.keyType.endsWith('_main') ? 'success' : 'info'}>{$t(keyTypeLabelKey(row.keyType))}</NTag>;
        if (isMainKeyType(row.keyType) || !row.scenarioName) return tag;
        return (
          <div class="flex flex-col items-center gap-2px">
            {tag}
            <span class="text-12px text-slate-400">{row.scenarioName}</span>
          </div>
        );
      }
    },
    {
      key: 'keyPrefix',
      title: $t('page.gateway.aiKey.col.keyPrefix'),
      align: 'center',
      minWidth: 600,
      // 默认只显前缀；hover 出眼睛/复制(触屏恒显)：眼睛切换明文(按需拉 value/:id 缓存行内)，复制取明文入剪贴板
      render: (row: AiKeyRow) => {
        const shown = row.keyRevealed && row.fullKeyValue ? row.fullKeyValue : row.keyPrefix;
        return (
          <div class="group flex items-center justify-center gap-4px">
            <span class="cursor-pointer font-mono text-13px break-all" onClick={() => toggleKeyReveal(row)}>
              {shown}
            </span>
            <div class="hidden items-center gap-2px group-hover:flex lt-sm:flex">
              <ButtonIcon
                text
                type="primary"
                class="h-24px text-14px"
                icon={row.keyRevealed ? 'material-symbols:visibility-off-outline' : 'material-symbols:visibility-outline'}
                tooltipContent={$t(row.keyRevealed ? 'page.gateway.aiKey.hideKey' : 'page.gateway.aiKey.viewKey')}
                onClick={() => toggleKeyReveal(row)}
              />
              <ButtonIcon
                text
                type="primary"
                class="h-24px text-14px"
                icon="material-symbols:content-copy-outline"
                tooltipContent={$t('page.gateway.aiKey.copyKey')}
                onClick={() => copyFullKey(row)}
              />
            </div>
          </div>
        );
      }
    },
    {
      key: 'owner',
      title: $t('page.gateway.aiKey.col.owner'),
      align: 'center',
      minWidth: 110,
      // 只显昵称/部门名(登录用户名已独立成列)；联表名缺失时兜底 ownerType:ownerId
      render: row => row.ownerName || `${row.ownerType}:${row.ownerId}`
    },
    {
      key: 'models',
      title: $t('page.gateway.aiKey.col.models'),
      align: 'center',
      minWidth: 110,
      // 资源=授权给该 Key 的可用能力：模型/MCP/Skill 三类计数，零授权项隐藏保持简洁
      render: row => {
        const counts: string[] = [];
        if ((row.models ?? []).length > 0) counts.push(`${$t('page.gateway.application.typeModel')} ${row.models!.length}`);
        if ((row.mcps ?? []).length > 0) counts.push(`MCP ${row.mcps!.length}`);
        if ((row.skills ?? []).length > 0) counts.push(`Skill ${row.skills!.length}`);
        if (counts.length === 0) return <span class="text-slate-400">-</span>;
        return <span class="text-13px">{counts.join(' · ')}</span>;
      }
    },
    {
      key: 'budget',
      title: $t('page.gateway.aiKey.col.budget'),
      align: 'center',
      minWidth: 160,
      render: row => {
        if (row.budgetLimit == null) {
          return <span class="text-slate-400">{$t('page.gateway.common.unlimited')}</span>;
        }
        const rate = row.budgetLimit > 0 ? Math.min((row.budgetUsed / row.budgetLimit) * 100, 100) : 0;
        return (
          <div class="flex flex-col items-center gap-2px">
            <NProgress type="line" percentage={rate} status={rate >= 85 ? 'error' : rate >= 60 ? 'warning' : 'success'} />
            <div class="flex items-center gap-4px">
              <span class="text-12px text-slate-400">{row.budgetUsed}/{row.budgetLimit}</span>
              <NTag size="small" bordered={false} type={row.budgetHardLimit ? 'error' : 'default'}>
                {$t(row.budgetHardLimit ? 'page.gateway.common.hardLimitOn' : 'page.gateway.common.hardLimitOff')}
              </NTag>
            </div>
          </div>
        );
      }
    },
    {
      key: 'rateLimit',
      title: $t('page.gateway.aiKey.col.rateLimit'),
      align: 'center',
      minWidth: 130,
      // none→灰字；total→TPM/RPM 值；per_model→按模型项数(明细在编辑抽屉维护)
      render: row => {
        if (!row.rateLimitMode || row.rateLimitMode === 'none') {
          return <span class="text-slate-400">{$t('page.gateway.common.rateLimitNone')}</span>;
        }
        if (row.rateLimitMode === 'total') {
          return (
            <div class="flex flex-col items-center gap-2px">
              <NTag size="small" bordered={false}>{$t('page.gateway.common.rateLimitTotal')}</NTag>
              <span class="text-12px text-slate-400">
                TPM {row.tpmLimit ?? '-'} / RPM {row.rpmLimit ?? '-'}
              </span>
            </div>
          );
        }
        return (
          <div class="flex flex-col items-center gap-2px">
            <NTag size="small" bordered={false}>{$t('page.gateway.common.rateLimitPerModel')}</NTag>
            <span class="text-12px text-slate-400">{$t('page.gateway.aiKey.modelCount', { n: Object.keys(row.modelLimits ?? {}).length })}</span>
          </div>
        );
      }
    },
    {
      key: 'isActive',
      title: $t('page.gateway.aiKey.col.isActive'),
      align: 'center',
      minWidth: 100,
      render: row => <NTag type={row.isActive ? 'success' : 'default'}>{$t(row.isActive ? 'page.gateway.common.active' : 'page.gateway.common.inactive')}</NTag>
    },
    {
      key: 'expiresAt',
      title: $t('page.gateway.aiKey.col.expiresAt'),
      align: 'center',
      minWidth: 110,
      // 过期由 LiteLLM expires_at 原生拦截；展示层：已过期红 / 7天内黄 / 永不过期灰
      render: row => {
        if (!row.expiresAt) return <span class="text-slate-400">{$t('page.gateway.aiKey.never')}</span>;
        const ts = Date.parse(row.expiresAt);
        const diff = ts - Date.now();
        if (diff < 0) {
          return (
            <NTag type="error">{$t('page.gateway.aiKey.expired')}</NTag>
          );
        }
        if (diff < 7 * 24 * 3600 * 1000) {
          return (
            <div class="flex flex-col items-center gap-2px">
              <NTag type="warning">{$t('page.gateway.aiKey.expiringSoon')}</NTag>
              <NTime time={ts} format="yyyy-MM-dd HH:mm" />
            </div>
          );
        }
        return <NTime time={ts} format="yyyy-MM-dd HH:mm" />;
      }
    },
    {
      key: 'lastUsedAt',
      title: $t('page.gateway.aiKey.col.lastUsedAt'),
      align: 'center',
      minWidth: 110,
      render: row => (row.lastUsedAt ? <NTime time={Date.parse(row.lastUsedAt)} format="yyyy-MM-dd HH:mm" /> : <span class="text-slate-400">-</span>)
    },
    {
      key: 'operate',
      title: $t('common.operate'),
      align: 'center',
      width: 200,
      render: row => {
        const editBtn = () => (
          <ButtonIcon
            text
            type="primary"
            icon="material-symbols:drive-file-rename-outline-outline"
            tooltipContent={$t('common.edit')}
            onClick={() => edit(row.aiKeyId!)}
          />
        );

        // 轮换：原地换 Key 值保归因；旧 Key 立即失效，新明文仅 owner 经 home 查看
        const rotateBtn = () => (
          <ButtonIcon
            text
            type="warning"
            icon="material-symbols:autorenew"
            tooltipContent={$t('page.gateway.aiKey.rotate')}
            popconfirmContent={$t('page.gateway.aiKey.rotateConfirm')}
            onPositiveClick={() => handleRotate(row.aiKeyId!)}
          />
        );

        // 以此为模板批量建场景 Key：预填该 Key 的授权/预算/限流配置(仅主 Key 行提供)
        const templateBtn = () =>
          isMainKeyType(row.keyType) ? (
            <ButtonIcon
              text
              type="info"
              icon="material-symbols:content-copy"
              tooltipContent={$t('page.gateway.aiKey.copyTemplate')}
              onClick={() => handleCopyTemplate(row)}
            />
          ) : null;

        const deleteBtn = () => (
          <ButtonIcon
            text
            type="error"
            icon="material-symbols:delete-outline"
            tooltipContent={$t('common.delete')}
            popconfirmContent={$t('common.confirmDelete')}
            onPositiveClick={() => handleDelete(row.aiKeyId!)}
          />
        );

        return (
          <div class="flex-center gap-8px">
            {editBtn()}
            {templateBtn()}
            {rotateBtn()}
            {deleteBtn()}
          </div>
        );
      }
    }
  ]
});

const { drawerVisible, operateType, editingData, handleAdd, handleEdit, checkedRowKeys, onBatchDeleted, onDeleted } =
  useTableOperate(data, 'aiKeyId', getData);

function edit(aiKeyId: CommonType.IdType) {
  handleEdit(aiKeyId);
}

async function handleBatchDelete() {
  const { error } = await fetchBatchDeleteAiKey(checkedRowKeys.value);
  if (error) return;
  onBatchDeleted();
}

async function handleDelete(aiKeyId: CommonType.IdType) {
  const { error } = await fetchBatchDeleteAiKey([aiKeyId]);
  if (error) return;
  onDeleted();
}

async function handleRotate(aiKeyId: CommonType.IdType) {
  const { error } = await fetchRotateAiKey(aiKeyId);
  if (error) return;
  window.$message?.success($t('page.gateway.aiKey.rotateSuccess'));
  getData();
}

/** 全量重推密钥投影到 LiteLLM(改名级联/授权对齐同步失败的漂移兜底) */
const resyncing = ref(false);

async function handleResync() {
  resyncing.value = true;
  const { data: res, error } = await fetchResyncAiKeys();
  resyncing.value = false;
  if (error) return;
  window.$message?.success(
    $t('page.gateway.aiKey.resyncSuccess', {
      pushed: res?.pushed ?? 0,
      skipped: res?.skipped ?? 0,
      failed: res?.failed?.length ?? 0
    })
  );
}

/** 批量开通/批量建场景 Key 弹窗(管理员效率件；templateRow 非空=复制主 Key 模板模式) */
const batchVisible = ref(false);
const batchInitialMode = ref<'main' | 'scene'>('main');
const batchTemplateRow = ref<Api.Gateway.AiKey | null>(null);

function handleCopyTemplate(row: AiKeyRow) {
  batchInitialMode.value = 'scene';
  batchTemplateRow.value = row;
  batchVisible.value = true;
}

// 场景管理侧变更(增/改/删场景)会影响密钥列表的场景列展示，由父组件调用 refresh 联动刷新
defineExpose({
  refresh: getData
});
</script>

<template>
  <div class="h-full flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <AiKeySearch v-model:model="searchParams" @reset="getDataByPage" @search="getDataByPage" />
    <NCard :title="$t('page.gateway.aiKey.tabKeys')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
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
        >
          <template #prefix>
            <NPopconfirm @positive-click="handleResync">
              <template #trigger>
                <NButton size="small" :loading="resyncing">
                  {{ $t('page.gateway.aiKey.resync') }}
                </NButton>
              </template>
              {{ $t('page.gateway.aiKey.resyncConfirm') }}
            </NPopconfirm>
            <NButton
              size="small"
              type="primary"
              ghost
              @click="
                () => {
                  batchInitialMode = 'main';
                  batchTemplateRow = null;
                  batchVisible = true;
                }
              "
            >
              {{ $t('page.gateway.aiKey.batchCreate') }}
            </NButton>
          </template>
        </TableHeaderOperation>
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
        :row-key="row => row.aiKeyId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
    </NCard>
    <AiKeyOperateDrawer
      v-model:visible="drawerVisible"
      :operate-type="operateType"
      :row-data="editingData"
      @submitted="getData"
    />
    <AiKeyBatchModal
      v-model:visible="batchVisible"
      :initial-mode="batchInitialMode"
      :template-row="batchTemplateRow"
      @submitted="getData"
    />
  </div>
</template>

<style scoped></style>
