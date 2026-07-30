<script setup lang="tsx">
import { ref } from 'vue';
import { NDivider, NTag, NSwitch, NTime } from 'naive-ui';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import {
  fetchGetTimedTaskList,
  fetchDeleteTimedTask,
  fetchToggleTimedTask,
  fetchTriggerTimedTask
} from '@/service/api/system/timer';
import { useAppStore } from '@/store/modules/app';
import { $t } from '@/locales';
import ButtonIcon from '@/components/custom/button-icon.vue';
import TimerSearch from './modules/timer-search.vue';
import TimerOperateDrawer from './modules/timer-operate-drawer.vue';
import TimerLogDrawer from './modules/timer-log-drawer.vue';

defineOptions({
  name: 'TimerList'
});

const appStore = useAppStore();

const searchParams = ref<Api.System.SysTimedTaskSearchParams>({
  pageNum: 1,
  pageSize: 10
});

/** 构建干净查询参数: 剔除 null/空字符串, 避免 axios 将 null 序列化为空字符串传给后端,
 *  防止 Gin 把 enabled= 空字符串绑定为 *bool=false, 导致误过滤 enabled=true 的任务 */
function buildCleanParams(): Api.System.SysTimedTaskSearchParams {
  const p: Partial<Api.System.SysTimedTaskSearchParams> = {
    pageNum: searchParams.value.pageNum,
    pageSize: searchParams.value.pageSize
  };
  if (searchParams.value.name) p.name = searchParams.value.name;
  if (searchParams.value.executorType) p.executorType = searchParams.value.executorType;
  if (searchParams.value.enabled !== null && searchParams.value.enabled !== undefined) {
    p.enabled = searchParams.value.enabled;
  }
  return p;
}

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } =
  useNaivePaginatedTable({
    api: () => fetchGetTimedTaskList(buildCleanParams()),
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
        key: 'id',
        title: $t('page.system.timer.id'),
        align: 'center',
        width: 80
      },
      {
        key: 'name',
        title: $t('page.system.timer.name'),
        align: 'center',
        minWidth: 140,
        ellipsis: {
          tooltip: true
        }
      },
      {
        key: 'description',
        title: $t('page.system.timer.description'),
        align: 'center',
        minWidth: 160,
        ellipsis: {
          tooltip: true
        }
      },
      {
        key: 'spec',
        title: $t('page.system.timer.spec'),
        align: 'center',
        width: 130
      },
      {
        key: 'executorType',
        title: $t('page.system.timer.executorType'),
        align: 'center',
        width: 100,
        render(row) {
          const typeInfo: Record<string, { type: 'info' | 'warning'; label: string }> = {
            method: { type: 'info', label: $t('page.system.timer.executorMethod') },
            http: { type: 'warning', label: $t('page.system.timer.executorHttp') }
          };
          const info = typeInfo[row.executorType] || { type: 'info' as const, label: row.executorType };
          return <NTag type={info.type} bordered={false} size="small">{info.label}</NTag>;
        }
      },
      {
        key: 'enabled',
        title: $t('page.system.timer.enabled'),
        align: 'center',
        width: 90,
        render(row) {
          return (
            <NSwitch
              value={row.enabled}
              onUpdateValue={(val: boolean) => handleToggle(row, val)}
            />
          );
        }
      },
      {
        key: 'nextRunAt',
        title: $t('page.system.timer.nextRunAt'),
        align: 'center',
        width: 170,
        render: row => {
          if (!row.nextRunAt) return '—';
          return <NTime time={Date.parse(row.nextRunAt)} format="yyyy-MM-dd HH:mm:ss" />;
        }
      },
      {
        key: 'createTime',
        title: $t('page.system.timer.createTime'),
        align: 'center',
        minWidth: 170,
        render: row => <NTime time={Date.parse(row.createTime)} format="yyyy-MM-dd HH:mm:ss" />
      },
      {
        key: 'operate',
        title: $t('common.operate'),
        align: 'center',
        width: 260,
        fixed: 'right',
        render: row => {
          const divider = () => <NDivider vertical />;

          const triggerBtn = () => (
            <ButtonIcon
              type="primary"
              text
              icon="material-symbols:play-arrow-outline"
              tooltipContent={$t('page.system.timer.trigger')}
              popconfirmContent={$t('page.system.timer.triggerConfirm')}
              onPositiveClick={() => handleTrigger(row)}
            />
          );

          const logBtn = () => (
            <ButtonIcon
              text
              icon="material-symbols:article-outline"
              tooltipContent={$t('page.system.timer.log')}
              onClick={() => openLogs(row)}
            />
          );

          const editBtn = () => (
            <ButtonIcon
              type="primary"
              text
              icon="material-symbols:drive-file-rename-outline-outline"
              tooltipContent={$t('common.edit')}
              onClick={() => edit(row.id!)}
            />
          );

          const deleteBtn = () => (
            <ButtonIcon
              text
              type="error"
              icon="material-symbols:delete-outline"
              tooltipContent={$t('common.delete')}
              popconfirmContent={$t('page.system.timer.deleteConfirm')}
              onPositiveClick={() => handleDelete(row.id!)}
            />
          );

          return (
            <div class="flex-center gap-8px">
              {triggerBtn()}
              {divider()}
              {logBtn()}
              {divider()}
              {editBtn()}
              {divider()}
              {deleteBtn()}
            </div>
          );
        }
      }
    ]
  });

const { drawerVisible, operateType, editingData, handleAdd, handleEdit, checkedRowKeys, onBatchDeleted, onDeleted } =
  useTableOperate(data, 'id', getData);

async function handleBatchDelete() {
  for (const id of checkedRowKeys.value) {
    const { error } = await fetchDeleteTimedTask({ ID: String(id) });
    if (error) return;
  }
  onBatchDeleted();
}

async function handleDelete(id: CommonType.IdType) {
  const { error } = await fetchDeleteTimedTask({ ID: String(id) });
  if (error) return;
  onDeleted();
}

async function edit(id: CommonType.IdType) {
  handleEdit(id);
}

async function handleToggle(row: Api.System.SysTimedTask, enabled: boolean) {
  const { error } = await fetchToggleTimedTask({ ID: String(row.id), enabled });
  if (error) return;
  window.$message?.success(enabled ? $t('page.system.timer.toggleSuccessOn') : $t('page.system.timer.toggleSuccessOff'));
  getData();
}

async function handleTrigger(row: Api.System.SysTimedTask) {
  const { error } = await fetchTriggerTimedTask({ ID: String(row.id) });
  if (error) return;
  window.$message?.success($t('page.system.timer.triggerSuccess'));
}

// Log drawer state
const logDrawerVisible = ref(false);
const logTaskId = ref(0);
const logTaskName = ref('');

function openLogs(row: Api.System.SysTimedTask) {
  logTaskId.value = Number(row.id);
  logTaskName.value = row.name;
  logDrawerVisible.value = true;
}

function handleResetSearch() {
  getDataByPage();
}
</script>

<template>
  <div class="h-full flex-col-stretch gap-12px overflow-hidden lt-sm:overflow-auto">
    <TimerSearch v-model:model="searchParams" @reset="handleResetSearch" @search="getDataByPage" />
    <NCard
      :title="$t('page.system.timer.listTitle')"
      :bordered="false"
      size="small"
      class="card-wrapper sm:flex-1-hidden"
    >
      <template #header-extra>
        <TableHeaderOperation
          v-model:columns="columnChecks"
          :disabled-delete="checkedRowKeys.length === 0"
          :loading="loading"
          :show-add="true"
          :show-delete="true"
          :show-export="false"
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
        :row-key="row => row.id"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
      <TimerOperateDrawer
        v-model:visible="drawerVisible"
        :operate-type="operateType"
        :row-data="editingData"
        @submitted="getData"
      />
      <TimerLogDrawer v-model:visible="logDrawerVisible" :task-id="logTaskId" :task-name="logTaskName" />
    </NCard>
  </div>
</template>

<style scoped>
:deep(.n-data-table-wrapper),
:deep(.n-data-table-base-table),
:deep(.n-data-table-base-table-body),
:deep(.n-data-table-empty) {
  height: 100%;
}

@media screen and (max-width: 800px) {
  :deep(.n-data-table-base-table-body) {
    max-height: calc(100vh - 400px - var(--calc-footer-height, 0px));
  }
}

@media screen and (max-width: 802px) {
  :deep(.n-data-table-base-table-body) {
    max-height: calc(100vh - 473px - var(--calc-footer-height, 0px));
  }
}

:deep(.n-card-header__main) {
  min-width: 69px !important;
}
</style>
