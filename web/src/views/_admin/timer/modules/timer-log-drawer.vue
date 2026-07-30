<script setup lang="tsx">
import { ref, watch } from 'vue';
import { NTag, NTime } from 'naive-ui';
import { fetchGetTimedTaskLogList } from '@/service/api/system/timer';
import { $t } from '@/locales';

defineOptions({
  name: 'TimerLogDrawer'
});

interface Props {
  taskId: number;
}

const props = defineProps<Props>();

const visible = defineModel<boolean>('visible', {
  default: false
});

const logData = ref<Api.System.SysTimedTaskLog[]>([]);
const logTotal = ref(0);
const logPage = ref(1);
const logPageSize = ref(10);
const logLoading = ref(false);

async function loadLogs() {
  logLoading.value = true;
  const { data, error } = await fetchGetTimedTaskLogList({
    pageNum: logPage.value,
    pageSize: logPageSize.value,
    taskId: props.taskId
  });
  logLoading.value = false;
  if (error) return;
  if (data) {
    logData.value = data.rows || [];
    logTotal.value = data.total || 0;
  }
}

function handleLogPageChange(page: number) {
  logPage.value = page;
  loadLogs();
}

const logColumns = [
  {
    type: 'expand' as const,
    renderExpand: (row: Api.System.SysTimedTaskLog) => {
      const hasDetail = row.errorMsg || row.output;
      if (!hasDetail) {
        return <div class="px-16px py-8px text-gray-400">{ $t('page.system.timer.noDetail') }</div>;
      }
      return (
        <div class="px-16px py-8px">
          {row.errorMsg ? <div class="text-red-500 break-all mb-8px">{ $t('page.system.timer.errorMsg') }：{row.errorMsg}</div> : null}
          {row.output ? <div class="break-all">{ $t('page.system.timer.output') }：{row.output}</div> : null}
        </div>
      );
    }
  },
  {
    key: 'triggerType',
    title: $t('page.system.timer.triggerType'),
    align: 'center' as const,
    width: 100,
    render(row: Api.System.SysTimedTaskLog) {
      return (
        <NTag type={row.triggerType === 'auto' ? 'info' : 'warning'} bordered={false} size="small">
          {row.triggerType === 'auto' ? $t('page.system.timer.triggerAuto') : $t('page.system.timer.triggerManual')}
        </NTag>
      );
    }
  },
  {
    key: 'status',
    title: $t('page.system.timer.status'),
    align: 'center' as const,
    width: 100,
    render(row: Api.System.SysTimedTaskLog) {
      const typeMap: Record<string, 'success' | 'error' | 'warning'> = {
        success: 'success',
        fail: 'error',
        timeout: 'warning'
      };
      const labelMap: Record<string, string> = {
        success: $t('page.system.timer.statusSuccess'),
        fail: $t('page.system.timer.statusFail'),
        timeout: $t('page.system.timer.statusTimeout')
      };
      return (
        <NTag type={typeMap[row.status] || 'default'} bordered={false} size="small">
          {labelMap[row.status] || row.status}
        </NTag>
      );
    }
  },
  {
    key: 'startedAt',
    title: $t('page.system.timer.startedAt'),
    align: 'center' as const,
    width: 170,
    render: (row: Api.System.SysTimedTaskLog) => (
      <NTime time={Date.parse(row.startedAt)} format="yyyy-MM-dd HH:mm:ss" />
    )
  },
  {
    key: 'durationMs',
    title: $t('page.system.timer.durationMs'),
    align: 'center' as const,
    width: 100
  }
];

watch(visible, () => {
  if (visible.value) {
    logPage.value = 1;
    loadLogs();
  }
});
</script>

<template>
  <NDrawer
    v-model:show="visible"
    :title="$t('page.system.timer.logTitle')"
    display-directive="show"
    :width="900"
    class="max-w-90%"
  >
    <NDrawerContent :title="$t('page.system.timer.logTitle')" :native-scrollbar="false" closable>
      <NDataTable
        :columns="logColumns"
        :data="logData"
        size="small"
        :loading="logLoading"
        remote
        :row-key="row => row.id"
        :pagination="{
          page: logPage,
          pageSize: logPageSize,
          itemCount: logTotal,
          onChange: handleLogPageChange
        }"
      />
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped></style>
