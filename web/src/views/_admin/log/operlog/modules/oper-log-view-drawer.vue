<script setup lang="tsx">
import { NDescriptions, NDescriptionsItem, NTag } from 'naive-ui';
import { getRequestMethodTagType } from '@/utils/icon-tag-format';
import { $t } from '@/locales';
import DictTag from '@/components/custom/dict-tag.vue';

defineOptions({
  name: 'OperLogViewDrawer'
});

interface Props {
  /** the edit row data */
  rowData: Api.Log.OperLog | null;
}

const props = defineProps<Props>();
const visible = defineModel<boolean>('visible', {
  default: false
});

const title = $t('page.log.operlog.detail');

function closeDrawer() {
  visible.value = false;
}
</script>

<template>
  <NDrawer v-model:show="visible" :title="title" display-directive="show" :width="800" class="max-w-90%">
    <NDrawerContent :title="title" :native-scrollbar="false" closable>
      <NDescriptions label-class="min-w-100px" :column="1" size="small" bordered label-placement="left">
        <NDescriptionsItem :label="$t('page.log.operlog.operId')">{{ props.rowData?.operId }}</NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.operlog.status')">
          <DictTag size="small" :value="props.rowData?.status" dict-code="sys_common_status" />
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.operlog.title')">
          <NSpace>
            <NTag class="m-1" size="small" type="primary">{{ props.rowData?.title }}{{ $t('page.log.operlog.module') }}</NTag>
            <DictTag size="small" :value="props.rowData?.businessType" dict-code="sys_oper_type" />
          </NSpace>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.operlog.operInfo')">
          {{ props.rowData?.operName }} | {{ props.rowData?.deptName }} | {{ props.rowData?.operIp }} |
          {{ props.rowData?.operLocation }}
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.operlog.requestInfo')">
          <NSpace>
            <NTag size="small" :type="getRequestMethodTagType(props.rowData?.requestMethod ?? '')">
              {{ props.rowData?.requestMethod?.toUpperCase() }}{{ $t('page.log.operlog.request') }}
            </NTag>
            {{ props.rowData?.operUrl }}
          </NSpace>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.operlog.operTime')">{{ props.rowData?.operTime }}</NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.operlog.operParam')">
          <JsonPreview :code="props.rowData?.operParam" />
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.operlog.jsonResult')">
          <JsonPreview :code="props.rowData?.jsonResult" />
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.operlog.costTime')">
          {{ `${props.rowData?.costTime} ms` }}
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.operlog.errorMsg')">{{ props.rowData?.errorMsg }}</NDescriptionsItem>
      </NDescriptions>
      <template #footer>
        <NSpace :size="16">
          <NButton @click="closeDrawer">{{ $t('common.close') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>
