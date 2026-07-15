<script setup lang="tsx">
import { ref } from 'vue';
import { NDivider } from 'naive-ui';
import {
  fetchBatchDeleteLoginLog,
  fetchCleanLoginLog,
  fetchGetLoginLogList,
  fetchUnlockLoginLog
} from '@/service/api/log/login-log';
import { useAppStore } from '@/store/modules/app';
import { useAuth } from '@/hooks/business/auth';
import { useDownload } from '@/hooks/business/download';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { useDict } from '@/hooks/business/dict';
import { getBrowserIcon, getOsIcon } from '@/utils/icon-tag-format';
import DictTag from '@/components/custom/dict-tag.vue';
import SvgIcon from '@/components/custom/svg-icon.vue';
import { $t } from '@/locales';
import ButtonIcon from '@/components/custom/button-icon.vue';
import LoginLogSearch from './modules/login-log-search.vue';
import LoginLogViewDrawer from './modules/login-log-view-drawer.vue';

defineOptions({
  name: 'LoginLogList'
});

const appStore = useAppStore();
const { download } = useDownload();
const { hasAuth } = useAuth();

useDict('sys_common_status');
useDict('sys_device_type');

const searchParams = ref<Api.Log.LoginLogSearchParams>({
  pageNum: 1,
  pageSize: 10,
  userName: null,
  ipaddr: null,
  status: null,
  params: {}
});

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } =
  useNaivePaginatedTable({
    api: () => fetchGetLoginLogList(searchParams.value),
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
        key: 'userName',
        title: $t('page.log.loginlog.userName'),
        align: 'center',
        minWidth: 120
      },
      {
        key: 'deviceType',
        title: $t('page.log.loginlog.deviceType'),
        align: 'center',
        minWidth: 120,
        render: row => {
          return <DictTag size="small" value={row.deviceType} dict-code="sys_device_type" />;
        }
      },
      {
        key: 'ipaddr',
        title: $t('page.log.loginlog.ipaddr'),
        align: 'center',
        minWidth: 120
      },
      {
        key: 'loginLocation',
        title: $t('page.log.loginlog.loginLocation'),
        align: 'center',
        minWidth: 120
      },
      {
        key: 'browser',
        title: $t('page.log.loginlog.browser'),
        align: 'center',
        minWidth: 120,
        render: row => {
          return (
            <div class="flex items-center justify-center gap-2">
              <SvgIcon icon={getBrowserIcon(row.browser)} />
              {row.browser}
            </div>
          );
        }
      },
      {
        key: 'os',
        title: $t('page.log.loginlog.os'),
        align: 'center',
        ellipsis: {
          tooltip: true
        },
        minWidth: 120,
        render: row => {
          const osName = row.os?.split(' or ')[0] ?? '';
          return (
            <div class="flex items-center justify-center gap-2">
              <SvgIcon icon={getOsIcon(osName)} />
              {osName}
            </div>
          );
        }
      },
      {
        key: 'status',
        title: $t('page.log.loginlog.status'),
        align: 'center',
        minWidth: 120,
        render: row => {
          return <DictTag size="small" value={row.status} dict-code="sys_common_status" />;
        }
      },
      {
        key: 'loginTime',
        title: $t('page.log.loginlog.loginTime'),
        align: 'center',
        ellipsis: {
          tooltip: true
        },
        minWidth: 120
      },
      {
        key: 'operate',
        title: $t('common.operate'),
        align: 'center',
        width: 130,
        render: row => {
          const viewBtn = () => {
            return (
              <ButtonIcon
                type="primary"
                text
                icon="material-symbols:visibility-outline"
                tooltipContent={$t('page.log.loginlog.view')}
                onClick={() => view(row.infoId!)}
              />
            );
          };

          const unlockBtn = () => {
            return (
              <>
                <NDivider vertical />
                <ButtonIcon
                  type="primary"
                  text
                  icon="material-symbols:lock-open-outline"
                  tooltipContent={$t('page.log.loginlog.unlock')}
                  popconfirmContent={$t('page.log.loginlog.confirmUnlock', { userName: row.userName })}
                  onPositiveClick={() => handleUnlockLoginLog(row.userName!)}
                />
              </>
            );
          };
          return (
            <div class="flex-center gap-8px">
              {viewBtn()}
              {unlockBtn()}
            </div>
          );
        }
      }
    ]
  });

const { drawerVisible, editingData, handleEdit, checkedRowKeys, onBatchDeleted } = useTableOperate(
  data,
  'infoId',
  getData
);

async function handleBatchDelete() {
  // request
  const { error } = await fetchBatchDeleteLoginLog(checkedRowKeys.value);
  if (error) return;
  onBatchDeleted();
}

async function view(infoId: CommonType.IdType) {
  handleEdit(infoId);
}

async function handleExport() {
  download('/monitor/loginlog/export', searchParams.value, `${$t('page.log.loginlog.exportFileName')}_${new Date().getTime()}.xlsx`);
}

async function handleCleanLoginLog() {
  window.$dialog?.error({
    title: $t('common.tip'),
    content: $t('page.log.loginlog.confirmClean'),
    positiveText: $t('page.log.loginlog.confirmCleanButton'),
    negativeText: $t('common.cancel'),
    onPositiveClick: async () => {
      const { error } = await fetchCleanLoginLog();
      if (error) return;
      window.$message?.success($t('page.log.loginlog.cleanSuccess'));
      await getData();
    }
  });
}

async function handleUnlockLoginLog(username: string) {
  const { error } = await fetchUnlockLoginLog(username);
  if (error) return;
  window.$message?.success($t('page.log.loginlog.unlockSuccess'));
  await getDataByPage();
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <LoginLogSearch v-model:model="searchParams" @search="getDataByPage" />
    <NCard :title="$t('page.log.loginlog.listTitle')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
      <template #header-extra>
        <TableHeaderOperation
          v-model:columns="columnChecks"
          :disabled-delete="checkedRowKeys.length === 0"
          :loading="loading"
          :show-add="false"
          :show-delete="hasAuth('log:loginlog:remove')"
          :show-export="hasAuth('log:loginlog:export')"
          @delete="handleBatchDelete"
          @export="handleExport"
          @refresh="getData"
        >
          <template #prefix>
            <NButton
              v-if="hasAuth('log:loginlog:remove')"
              type="error"
              ghost
              size="small"
              @click="handleCleanLoginLog"
            >
              <template #icon>
                <icon-material-symbols-warning-outline-rounded />
              </template>
              {{ $t('page.log.loginlog.clean') }}
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
        :row-key="row => row.infoId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
      <LoginLogViewDrawer v-model:visible="drawerVisible" :row-data="editingData" />
    </NCard>
  </div>
</template>

<style scoped></style>
