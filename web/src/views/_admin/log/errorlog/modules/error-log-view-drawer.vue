<script setup lang="tsx">
import { NTag, NTime } from 'naive-ui';
import { $t } from '@/locales';

defineOptions({
  name: 'ErrorLogViewDrawer'
});

interface Props {
  rowData: Api.Log.ErrorLog | null;
}

const props = defineProps<Props>();
const visible = defineModel<boolean>('visible', {
  default: false
});

const title = $t('page.log.errorlog.detail');

function closeDrawer() {
  visible.value = false;
}

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
</script>

<template>
  <NDrawer v-model:show="visible" :title="title" display-directive="show" :width="900" class="max-w-90%">
    <NDrawerContent :title="title" :native-scrollbar="false" closable>
      <NDescriptions label-class="min-w-100px" :column="1" size="small" bordered label-placement="left">
        <NDescriptionsItem :label="$t('page.log.errorlog.form')">
          {{ props.rowData?.form || '-' }}
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.errorlog.level')">
          <NTag :type="levelTagMap[props.rowData?.level ?? ''] || 'info'" size="small">
            {{ levelLabelMap[props.rowData?.level ?? ''] || '一般错误' }}
          </NTag>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.errorlog.status')">
          <NTag :type="statusTagMap[props.rowData?.status ?? ''] || 'info'" size="small">
            {{ props.rowData?.status || '未处理' }}
          </NTag>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.errorlog.requestId')">
          {{ props.rowData?.request_id || '-' }}
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.errorlog.traceId')">
          {{ props.rowData?.trace_id || '-' }}
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.errorlog.createTime')">
          <NTime v-if="props.rowData?.createTime" :time="Date.parse(props.rowData.createTime)" format="yyyy-MM-dd HH:mm:ss" />
          <template v-else>-</template>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.errorlog.info')">
          <pre class="whitespace-pre-wrap break-words max-h-400px overflow-auto text-13px">{{ props.rowData?.info || '-' }}</pre>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.errorlog.solution')">
          <pre class="whitespace-pre-wrap break-words max-h-400px overflow-auto text-13px">{{ props.rowData?.solution || '-' }}</pre>
        </NDescriptionsItem>
      </NDescriptions>
      <template #footer>
        <NSpace :size="16">
          <NButton @click="closeDrawer">{{ $t('common.close') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>
