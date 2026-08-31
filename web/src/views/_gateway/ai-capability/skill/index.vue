<script setup lang="tsx">
import { onMounted, ref } from 'vue';
import { NTag } from 'naive-ui';
import { fetchBatchDeleteSkills, fetchGetSkillList } from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable } from '@/hooks/common/table';
import { $t } from '@/locales';
import SkillSearch from './modules/skill-search.vue';
import SkillOperateDrawer from './modules/skill-operate-drawer.vue';
import SkillUsageDrawer from './modules/skill-usage-drawer.vue';

defineOptions({ name: 'GatewaySkill' });

const appStore = useAppStore();

const searchParams = ref<Api.Gateway.SkillSearchParams>({
  pageNum: 1,
  pageSize: 20,
  name: null,
  category: null,
  isActive: null,
  isPublished: null,
  params: {}
});

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } =
  useNaivePaginatedTable({
    api: () => {
      const { pageNum, pageSize, name, category } = searchParams.value;
      const params: Api.Gateway.SkillSearchParams = { pageNum, pageSize };
      if (name) params.name = name;
      if (category) params.category = category;
      if (searchParams.value.isActive !== null && searchParams.value.isActive !== undefined) {
        params.isActive = searchParams.value.isActive;
      }
      if (searchParams.value.isPublished !== null && searchParams.value.isPublished !== undefined) {
        params.isPublished = searchParams.value.isPublished;
      }
      return fetchGetSkillList(params);
    },
    transform: response => defaultTransform(response),
    onPaginationParamsChange: params => {
      searchParams.value.pageNum = params.page;
      searchParams.value.pageSize = params.pageSize;
    },
    columns: () => [
      {
        key: 'name',
        title: $t('page.gateway.skill.col.name'),
        align: 'center',
        minWidth: 150,
        render: row => (
          <div class="flex flex-col items-start">
            <span class="font-500">{row.name}</span>
            <span class="text-12px text-slate-400">
              v{row.version} · {row.author || '-'}
            </span>
          </div>
        )
      },
      {
        key: 'category',
        title: $t('page.gateway.skill.col.category'),
        align: 'center',
        minWidth: 90,
        render: row => <NTag size="small">{row.category}</NTag>
      },
      {
        key: 'tags',
        title: $t('page.gateway.skill.col.tags'),
        align: 'center',
        minWidth: 140,
        render: row =>
          row.tags && row.tags.length > 0 ? (
            <div class="flex flex-wrap justify-center gap-4px">
              {row.tags.slice(0, 3).map(t => (
                <NTag size="small" type="info" bordered={false}>
                  {t}
                </NTag>
              ))}
            </div>
          ) : (
            <span class="text-slate-400">-</span>
          )
      },
      {
        key: 'zipFilename',
        title: $t('page.gateway.skill.col.zipPackage'),
        align: 'center',
        minWidth: 110,
        render: row =>
          row.zipFilename ? (
            <NTag size="small" type="success">
              {formatSize(row.zipSize || 0)}
            </NTag>
          ) : (
            <NTag size="small" type="warning">
              {$t('page.gateway.skill.noPackage')}
            </NTag>
          )
      },
      {
        key: 'installCount',
        title: $t('page.gateway.skill.col.installCount'),
        align: 'center',
        minWidth: 90
      },
      {
        key: 'isPublished',
        title: $t('page.gateway.skill.col.isPublished'),
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
        title: $t('page.gateway.skill.col.isActive'),
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
        key: 'actions',
        title: $t('common.action'),
        align: 'center',
        width: 160,
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
              onClick={() => handleUsage(row)}
            >
              {$t('page.gateway.skill.usage.title')}
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

function formatSize(size: number): string {
  if (!size) return '0 B';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}

// 新增/编辑抽屉
const drawerVisible = ref(false);
const operateType = ref<NaiveUI.TableOperateType>('add');
const editingData = ref<Api.Gateway.Skill | null>(null);

function handleAdd() {
  operateType.value = 'add';
  editingData.value = null;
  drawerVisible.value = true;
}
function handleEdit(row: Api.Gateway.Skill) {
  operateType.value = 'edit';
  editingData.value = row;
  drawerVisible.value = true;
}
async function handleDelete(row: Api.Gateway.Skill) {
  const { error } = await fetchBatchDeleteSkills([row.skillId!]);
  if (error) return;
  window.$message?.success($t('common.deleteSuccess'));
  getData();
}
function handleSubmitted() {
  drawerVisible.value = false;
  getData();
}

// 使用日志抽屉
const usageVisible = ref(false);
const usageRow = ref<Api.Gateway.Skill | null>(null);

function handleUsage(row: Api.Gateway.Skill) {
  usageRow.value = row;
  usageVisible.value = true;
}

onMounted(() => {
  getData();
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <SkillSearch v-model:model="searchParams" @search="getDataByPage" />

    <NCard :title="$t('page.gateway.skill.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
      <template #header-extra>
        <NSpace size="small">
          <NButton size="small" type="primary" @click="handleAdd">
            {{ $t('page.gateway.skill.add') }}
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
        :row-key="row => row.skillId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
    </NCard>

    <SkillOperateDrawer
      v-model:visible="drawerVisible"
      :operate-type="operateType"
      :row-data="editingData"
      @submitted="handleSubmitted"
    />
    <SkillUsageDrawer v-model:visible="usageVisible" :row="usageRow" />
  </div>
</template>

<style scoped></style>
