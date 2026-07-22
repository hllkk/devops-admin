<script setup lang="tsx">
import { ref, watch } from 'vue';
import { NTag, NTime } from 'naive-ui';
import {
  fetchBatchDeleteErrorLog,
  fetchDeleteErrorLog,
  fetchGetErrorLogDetail,
  fetchGetErrorLogList,
  fetchGetErrorLogSolution
} from '@/service/api/log';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { $t } from '@/locales';
import { errorlogUpdateEvent } from '@/utils/sse';
import ButtonIcon from '@/components/custom/button-icon.vue';
import ErrorLogViewDrawer from './modules/error-log-view-drawer.vue';
import ErrorLogSearch from './modules/error-log-search.vue';

defineOptions({
  name: 'ErrorLogList'
});

const appStore = useAppStore();

const searchParams = ref<Api.Log.ErrorLogSearchParams>({
  pageNum: 1,
  pageSize: 10,
  form: null,
  info: null,
  level: null,
  status: null,
  createdAtRange: null
});

const levelTagMap: Record<string, 'warning' | 'error' | 'info'> = {
  error: 'warning',
  fatal: 'error'
};
const levelLabelMap: Record<string, string> = {
  error: '一般错误',
  fatal: '致命错误'
};
const statusTagMap: Record<string, 'info' | 'warning' | 'success' | 'error'> = {
  '未处理': 'info',
  '处理中': 'warning',
  '处理完成': 'success',
  '处理失败': 'error'
};

// 旋转动画内联样式(scoped CSS 不穿透 NaiveUI JSX render, 改用内联)
const spinnerStyle = {
  display: 'inline-block',
  width: '12px',
  height: '12px',
  border: '2px solid #e6a23c',
  borderTopColor: 'transparent',
  borderRadius: '50%',
  animation: 'errorlog-spin 0.8s linear infinite'
};

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } =
  useNaivePaginatedTable({
    api: () => fetchGetErrorLogList(searchParams.value),
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
        key: 'index',
        title: $t('common.index'),
        align: 'center',
        width: 64,
        render: (_, index) => index + 1
      },
      {
        key: 'createTime',
        title: $t('page.log.errorlog.createTime'),
        align: 'center',
        minWidth: 160,
        render: row => <NTime time={Date.parse(row.createTime)} format="yyyy-MM-dd HH:mm:ss" />
      },
      {
        key: 'form',
        title: $t('page.log.errorlog.form'),
        align: 'center',
        minWidth: 100
      },
      {
        key: 'level',
        title: $t('page.log.errorlog.level'),
        align: 'center',
        minWidth: 100,
        render(row) {
          return (
            <NTag size="small" type={levelTagMap[row.level] || 'info'}>
              {levelLabelMap[row.level] || '一般错误'}
            </NTag>
          );
        }
      },
      {
        key: 'status',
        title: $t('page.log.errorlog.status'),
        align: 'center',
        minWidth: 120,
        render(row) {
          if (row.status === '处理中') {
            return (
              <NTag size="small" type="warning">
                <span class="flex-center gap-4px">
                  <span style={spinnerStyle} />
                  处理中
                </span>
              </NTag>
            );
          }
          return (
            <NTag size="small" type={statusTagMap[row.status] || 'info'}>
              {row.status || '未处理'}
            </NTag>
          );
        }
      },
      {
        key: 'info',
        title: $t('page.log.errorlog.info'),
        align: 'left',
        minWidth: 200,
        maxWidth: 400,
        ellipsis: { tooltip: true }
      },
      {
        key: 'operate',
        title: $t('common.operate'),
        align: 'center',
        width: 190,
        fixed: 'right',
        render: row => {
          return (
            <div class="flex-center gap-8px">
              {row.status !== '处理中' && (
                <ButtonIcon
                  type="primary"
                  text
                  icon="material-symbols:auto-awesome"
                  tooltipContent={$t('page.log.errorlog.getSolution')}
                  onClick={() => getSolution(row.id)}
                />
              )}
              <ButtonIcon
                type="primary"
                text
                icon="material-symbols:visibility-outline"
                tooltipContent={$t('page.log.errorlog.view')}
                onClick={() => viewDetail(row.id)}
              />
              <ButtonIcon
                type="error"
                text
                icon="material-symbols:delete-outline"
                tooltipContent={$t('common.delete')}
                onClick={() => handleDeleteRow(row.id)}
              />
            </div>
          );
        }
      }
    ]
  });

const { checkedRowKeys, onBatchDeleted } = useTableOperate(data, 'id', getData);

// SSE 推送: 后端 AI 处理完成后自动刷新列表
watch(errorlogUpdateEvent, newVal => {
  if (newVal && newVal.id) {
    getData();
  }
});

// 查看详情
const detailDrawerVisible = ref(false);
const detailData = ref<Api.Log.ErrorLog | null>(null);

async function viewDetail(id: string) {
  const { data: result, error } = await fetchGetErrorLogDetail(id);
  if (error || !result) return;
  detailData.value = result;
  detailDrawerVisible.value = true;
}

// AI 方案
async function getSolution(id: string) {
  const confirmed = await new Promise<boolean>(resolve => {
    window.$dialog?.warning({
      title: $t('common.tip'),
      content: $t('page.log.errorlog.confirmGetSolution'),
      positiveText: $t('common.confirm'),
      negativeText: $t('common.cancel'),
      onPositiveClick: () => resolve(true),
      onNegativeClick: () => resolve(false),
      onClose: () => resolve(false)
    });
  });
  if (!confirmed) return;
  const { error } = await fetchGetErrorLogSolution(id);
  if (error) return;
  window.$message?.success($t('page.log.errorlog.solutionSubmitted'));
  await getData();
}

// 单行删除
async function handleDeleteRow(id: string) {
  window.$dialog?.warning({
    title: $t('common.tip'),
    content: $t('common.confirmDelete'),
    positiveText: $t('common.confirm'),
    negativeText: $t('common.cancel'),
    onPositiveClick: async () => {
      const { error } = await fetchDeleteErrorLog(id);
      if (error) return;
      window.$message?.success($t('common.deleteSuccess'));
      await getData();
    }
  });
}

// 批量删除
async function handleBatchDelete() {
  const ids = checkedRowKeys.value.map(String);
  const { error } = await fetchBatchDeleteErrorLog(ids);
  if (error) return;
  onBatchDeleted();
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <ErrorLogSearch v-model:model="searchParams" @search="getDataByPage" />
    <NCard
      :title="$t('page.log.errorlog.listTitle')"
      :bordered="false"
      size="small"
      class="card-wrapper sm:flex-1-hidden"
    >
      <template #header-extra>
        <TableHeaderOperation
          v-model:columns="columnChecks"
          :disabled-delete="checkedRowKeys.length === 0"
          :loading="loading"
          :show-add="false"
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
      <ErrorLogViewDrawer v-model:visible="detailDrawerVisible" :row-data="detailData" />
    </NCard>
  </div>
</template>

<style>
/* 全局 keyframes (scoped 不穿透 NaiveUI JSX render) */
@keyframes errorlog-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
