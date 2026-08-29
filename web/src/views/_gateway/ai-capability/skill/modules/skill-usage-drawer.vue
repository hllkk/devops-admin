<script setup lang="tsx">
import { ref, watch } from 'vue';
import { NTag } from 'naive-ui';
import { fetchGetSkillUsageList } from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable } from '@/hooks/common/table';
import { $t } from '@/locales';
import { SKILL_USAGE_ACTION_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'SkillUsageDrawer' });

interface Props {
  row: Api.Gateway.Skill | null;
}

const props = defineProps<Props>();

const visible = defineModel<boolean>('visible', {
  default: false
});

const appStore = useAppStore();

const paginationPage = ref(1);
const paginationSize = ref(10);

const { columns, data, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => {
    const params: Api.Gateway.SkillUsageSearchParams = { pageNum: paginationPage.value, pageSize: paginationSize.value };
    if (props.row?.skillId) params.skillId = props.row.skillId;
    return fetchGetSkillUsageList(params);
  },
  transform: response => defaultTransform(response),
  onPaginationParamsChange: params => {
    paginationPage.value = params.page ?? 1;
    paginationSize.value = params.pageSize ?? 10;
  },
  columns: () => [
    {
      key: 'createTime',
      title: $t('page.gateway.skill.usage.col.createTime'),
      align: 'center',
      minWidth: 160
    },
    {
      key: 'userName',
      title: $t('page.gateway.skill.usage.col.userName'),
      align: 'center',
      minWidth: 120
    },
    {
      key: 'skillName',
      title: $t('page.gateway.skill.usage.col.skillName'),
      align: 'center',
      minWidth: 140
    },
    {
      key: 'action',
      title: $t('page.gateway.skill.usage.col.action'),
      align: 'center',
      minWidth: 90,
      render: row => {
        const opt = SKILL_USAGE_ACTION_OPTIONS.find(o => o.value === row.action);
        return <NTag size="small">{opt ? $t(opt.label) : row.action}</NTag>;
      }
    }
  ]
});

watch(visible, val => {
  if (val) {
    paginationPage.value = 1;
    getDataByPage(1);
  }
});
</script>

<template>
  <NDrawer v-model:show="visible" display-directive="show" :width="620" class="max-w-90%">
    <NDrawerContent :title="$t('page.gateway.skill.usage.title')" :native-scrollbar="false" closable>
      <p class="mb-12px text-12px text-slate-400">{{ $t('page.gateway.skill.usage.tip') }}</p>
      <NDataTable
        :columns="columns"
        :data="data"
        size="small"
        :flex-height="!appStore.isMobile"
        :scroll-x="scrollX"
        :loading="loading"
        remote
        :row-key="row => row.id"
        :pagination="mobilePagination"
      />
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped></style>
