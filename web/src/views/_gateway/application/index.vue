<script setup lang="tsx">
import { onMounted, ref } from 'vue';
import { NButton, NTag, NTime } from 'naive-ui';
import type { DataTableRowKey } from 'naive-ui';
import {
  fetchApproveApplication,
  fetchBatchReviewApplications,
  fetchGetApplicationList,
  fetchRejectApplication
} from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable } from '@/hooks/common/table';
import { $t } from '@/locales';
import ApplicationSearch from './modules/application-search.vue';

defineOptions({ name: 'GatewayApplication' });

const appStore = useAppStore();

/** 状态筛选默认 pending(审批列表主视角;空串=全部) */
const searchParams = ref<Api.Gateway.ApplicationSearchParams>({
  pageNum: 1,
  pageSize: 20,
  status: 'pending',
  resourceType: null,
  userId: null,
  params: {}
});

const APP_STATUS_META: Record<
  Api.Gateway.ApplicationItem['status'],
  { label: 'statusPending' | 'statusApproved' | 'statusRejected'; type: 'warning' | 'success' | 'error' }
> = {
  pending: { label: 'statusPending', type: 'warning' },
  approved: { label: 'statusApproved', type: 'success' },
  rejected: { label: 'statusRejected', type: 'error' }
};

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => {
    const { pageNum, pageSize, status, resourceType, userId } = searchParams.value;
    const params: Api.Gateway.ApplicationSearchParams = { pageNum, pageSize };
    if (status) params.status = status;
    if (resourceType) params.resourceType = resourceType;
    if (userId) params.userId = userId;
    return fetchGetApplicationList(params);
  },
  transform: response => defaultTransform(response),
  onPaginationParamsChange: params => {
    searchParams.value.pageNum = params.page;
    searchParams.value.pageSize = params.pageSize;
  },
  columns: () => [
    {
      key: 'userName',
      title: $t('page.gateway.application.col.applicant'),
      align: 'center',
      minWidth: 110,
      render: row => row.userName || '-'
    },
    {
      key: 'resourceName',
      title: $t('page.gateway.application.col.resource'),
      align: 'center',
      minWidth: 180,
      render: row => (
        <div class="flex flex-col items-center">
          <span>{row.resourceName || '-'}</span>
          <code class="text-12px text-slate-400">{row.resourceKey}</code>
        </div>
      )
    },
    {
      key: 'resourceType',
      title: $t('page.gateway.application.col.resourceType'),
      align: 'center',
      minWidth: 80,
      render: row => (
        <NTag size="small" bordered={false}>
          {$t(row.resourceType === 'mcp' ? 'page.gateway.application.typeMcp' : 'page.gateway.application.typeModel')}
        </NTag>
      )
    },
    {
      key: 'reason',
      title: $t('page.gateway.application.col.reason'),
      align: 'center',
      minWidth: 160,
      ellipsis: { tooltip: true },
      render: row => row.reason || <span class="text-slate-400">-</span>
    },
    {
      key: 'status',
      title: $t('page.gateway.application.col.status'),
      align: 'center',
      minWidth: 90,
      render: row => (
        <NTag size="small" type={APP_STATUS_META[row.status].type} bordered={false}>
          {$t(`page.gateway.application.${APP_STATUS_META[row.status].label}`)}
        </NTag>
      )
    },
    {
      key: 'createTime',
      title: $t('page.gateway.application.col.applyTime'),
      align: 'center',
      minWidth: 160,
      render: row => <NTime time={new Date(row.createTime)} type="datetime" />
    },
    {
      key: 'reviewerName',
      title: $t('page.gateway.application.col.reviewer'),
      align: 'center',
      minWidth: 110,
      render: row => row.reviewerName || <span class="text-slate-400">-</span>
    },
    {
      key: 'operate',
      title: $t('page.gateway.application.col.action'),
      align: 'center',
      width: 140,
      render: row =>
        row.status === 'pending' ? (
          <div class="flex justify-center gap-6px">
            <NButton size="tiny" type="primary" onClick={() => openReview('approve', [row.applicationId])}>
              {$t('page.gateway.application.reviewApprove')}
            </NButton>
            <NButton size="tiny" type="error" ghost onClick={() => openReview('reject', [row.applicationId])}>
              {$t('page.gateway.application.reviewReject')}
            </NButton>
          </div>
        ) : (
          <span class="text-slate-400">-</span>
        )
    }
  ]
});

// ===== 勾选与批量审批 =====
const checkedKeys = ref<DataTableRowKey[]>([]);

/** 勾选行转申请ID列表(模板内不做类型断言) */
function checkedIds() {
  return checkedKeys.value as CommonType.IdType[];
}

// ===== 审批弹窗(单条/批量共用;通过与驳回仅差 mode) =====
const reviewVisible = ref(false);
const reviewMode = ref<'approve' | 'reject'>('approve');
const reviewTargets = ref<CommonType.IdType[]>([]);
const reviewNotes = ref('');
const reviewSubmitting = ref(false);

function openReview(mode: 'approve' | 'reject', ids: CommonType.IdType[]) {
  reviewMode.value = mode;
  reviewTargets.value = ids;
  reviewNotes.value = '';
  reviewVisible.value = true;
}

async function handleReviewSubmit() {
  if (!reviewTargets.value.length) return;
  reviewSubmitting.value = true;
  const approve = reviewMode.value === 'approve';
  const notes = reviewNotes.value.trim();
  const res =
    reviewTargets.value.length === 1
      ? await (approve
          ? fetchApproveApplication({ applicationId: reviewTargets.value[0], reviewNotes: notes })
          : fetchRejectApplication({ applicationId: reviewTargets.value[0], reviewNotes: notes }))
      : await fetchBatchReviewApplications({ applicationIds: reviewTargets.value, reviewNotes: notes }, approve);
  reviewSubmitting.value = false;
  if (res.error) return;
  // 批量结果:有失败项给警告汇总;单条 approve 带同步警告时提示(每日密钥重同步兜底)
  if (res.data && 'failed' in res.data) {
    if (res.data.failed.length) {
      window.$message?.warning(
        $t('page.gateway.application.batchResult', { success: res.data.success.length, failed: res.data.failed.length })
      );
    } else {
      window.$message?.success(
        $t('page.gateway.application.batchResult', { success: res.data.success.length, failed: 0 })
      );
    }
  } else if (approve && res.data?.warnings.length) {
    window.$message?.warning($t('page.gateway.application.reviewWarning'));
  } else {
    window.$message?.success(
      approve ? $t('page.gateway.application.approveSuccess') : $t('page.gateway.application.rejectSuccess')
    );
  }
  reviewVisible.value = false;
  checkedKeys.value = [];
  getData();
}

onMounted(() => {
  getData();
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <ApplicationSearch v-model:model="searchParams" @search="getDataByPage" />

    <NCard :title="$t('page.gateway.application.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
      <template #header-extra>
        <NSpace size="small">
          <NButton
            size="small"
            type="primary"
            :disabled="!checkedKeys.length"
            @click="openReview('approve', checkedIds())"
          >
            {{ $t('page.gateway.application.batchApprove') }}({{ checkedKeys.length }})
          </NButton>
          <NButton
            size="small"
            type="error"
            ghost
            :disabled="!checkedKeys.length"
            @click="openReview('reject', checkedIds())"
          >
            {{ $t('page.gateway.application.batchReject') }}({{ checkedKeys.length }})
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
      <p class="mb-8px text-12px text-slate-400">{{ $t('page.gateway.application.subtitle') }}</p>
      <NDataTable
        v-model:checked-row-keys="checkedKeys"
        :columns="columns"
        :data="data"
        size="small"
        :flex-height="!appStore.isMobile"
        :scroll-x="scrollX"
        :loading="loading"
        remote
        :row-key="row => row.applicationId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
    </NCard>

    <!-- 审批弹窗:意见可选,通过后自动授权到申请人个人主 Key -->
    <NModal
      v-model:show="reviewVisible"
      :title="
        reviewMode === 'approve'
          ? $t('page.gateway.application.batchApprove')
          : $t('page.gateway.application.batchReject')
      "
      preset="card"
      class="w-480px max-w-90%"
      :mask-closable="false"
    >
      <div class="flex flex-col gap-14px">
        <p class="text-13px text-slate-500 dark:text-slate-400">
          {{
            reviewMode === 'approve'
              ? $t('page.gateway.application.approveConfirm')
              : $t('page.gateway.application.rejectConfirm')
          }}
          ({{ reviewTargets.length }})
        </p>
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.gateway.application.reviewNotes') }}</div>
          <NInput
            v-model:value="reviewNotes"
            type="textarea"
            :rows="3"
            maxlength="500"
            show-count
            :placeholder="$t('page.gateway.application.reviewNotesPlaceholder')"
          />
        </div>
        <div class="flex justify-end gap-8px">
          <NButton quaternary @click="reviewVisible = false">{{ $t('common.cancel') }}</NButton>
          <NButton
            :type="reviewMode === 'approve' ? 'primary' : 'error'"
            :loading="reviewSubmitting"
            @click="handleReviewSubmit"
          >
            {{ $t('common.confirm') }}
          </NButton>
        </div>
      </div>
    </NModal>
  </div>
</template>

<style scoped></style>
