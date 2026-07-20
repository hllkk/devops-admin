<script setup lang="ts">
import { getBrowserIcon, getOsIcon } from '@/utils/icon-tag-format';
import { $t } from '@/locales';

defineOptions({
  name: 'LoginLogViewDrawer'
});

interface Props {
  /** the edit row data */
  rowData: Api.Log.LoginLog | null;
}



const props = defineProps<Props>();

const visible = defineModel<boolean>('visible', {
  default: false
});

const title = $t('page.log.loginlog.detail');

function closeDrawer() {
  visible.value = false;
}
</script>

<template>
  <NDrawer v-model:show="visible" :title="title" display-directive="show" :width="800" class="max-w-90%">
    <NDrawerContent :title="title" :native-scrollbar="false" closable>
      <NDescriptions label-placement="left" :column="1" size="small" bordered>
        <NDescriptionsItem :label="$t('page.log.loginlog.accountInfo')">
          {{ props.rowData?.userName }} | {{ props.rowData?.ipaddr }} | {{ props.rowData?.loginLocation }}
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.loginlog.client')">
          {{ props.rowData?.clientKey }}
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.loginlog.deviceType')">
          <DictTag size="small" :value="props.rowData?.deviceType" dict-code="sys_device_type" />
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.loginlog.browser')">
          <div class="flex items-center gap-2">
            <SvgIcon :icon="getBrowserIcon(props.rowData?.browser ?? '')" />
            {{ props.rowData?.browser }}
          </div>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.loginlog.os')">
          <div class="flex items-center gap-2">
            <SvgIcon :icon="getOsIcon(props.rowData?.os ?? '')" />
            {{ props.rowData?.os }}
          </div>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.loginlog.status')">
          <DictTag size="small" :value="props.rowData?.status" dict-code="sys_common_status" />
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.loginlog.msg')">
          {{ props.rowData?.msg }}
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.log.loginlog.loginTime')">
          <NTime :time="Date.parse(props.rowData?.loginTime ?? '')" format="yyyy-MM-dd HH:mm:ss" />
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

<style scoped></style>
