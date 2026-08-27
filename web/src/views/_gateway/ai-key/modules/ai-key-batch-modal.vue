<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { fetchGetUserSelect } from '@/service/api/system';
import { fetchBatchCreateMainKeys } from '@/service/api/gateway';
import { $t } from '@/locales';
import DeptTreeSelect from '@/components/custom/dept-tree-select.vue';

defineOptions({ name: 'AiKeyBatchModal' });

interface Emits {
  (e: 'submitted'): void;
}

const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', {
  default: false
});

/** 开通方式：按部门(取部门下全部用户) / 指定用户 */
const mode = ref<'dept' | 'users'>('dept');
const deptId = ref<CommonType.IdType | null>(null);
const userIds = ref<CommonType.IdType[]>([]);
const userOptions = ref<CommonType.Option<CommonType.IdType>[]>([]);
const submitting = ref(false);

/** 开通结果(null=未提交；部分失败语义：failed 空数组=全部成功) */
const result = ref<Api.Gateway.AiKeyBatchCreateResult | null>(null);

const modeOptions = computed(() => [
  { label: $t('page.gateway.aiKey.batchModeDept'), value: 'dept' },
  { label: $t('page.gateway.aiKey.batchModeUsers'), value: 'users' }
]);

async function loadUserOptions() {
  const { error, data } = await fetchGetUserSelect();
  if (!error && data) {
    userOptions.value = data.map(item => ({
      label: `${item.nickName} ( ${item.userName} )`,
      value: item.userId
    }));
  }
}

function resetForm() {
  mode.value = 'dept';
  deptId.value = null;
  userIds.value = [];
  result.value = null;
}

function handleClose() {
  // 已产生开通结果时，关闭即通知父组件刷新列表
  if (result.value) emit('submitted');
  visible.value = false;
}

async function handleSubmit() {
  if (mode.value === 'dept' && !deptId.value) {
    window.$message?.warning($t('page.gateway.aiKey.batchDeptRequired'));
    return;
  }
  if (mode.value === 'users' && userIds.value.length === 0) {
    window.$message?.warning($t('page.gateway.aiKey.batchUsersRequired'));
    return;
  }
  submitting.value = true;
  const { error, data } = await fetchBatchCreateMainKeys(
    mode.value === 'dept' ? { deptId: deptId.value } : { userIds: userIds.value }
  );
  submitting.value = false;
  if (error) return;
  // 防御规范化：契约是 failed 空数组=全部成功，但旧后端/异常数据可能给 null，
  // 直接 .length 会让渲染链 TypeError 崩溃(弹窗冻结关不掉)
  result.value = { ...data, failed: data.failed ?? [] };
  window.$message?.success(
    $t('page.gateway.aiKey.batchResult', {
      created: data.created,
      skipped: data.skipped,
      failed: result.value.failed.length
    })
  );
}

watch(visible, val => {
  if (val) {
    resetForm();
    if (userOptions.value.length === 0) loadUserOptions();
  }
});
</script>

<template>
  <NModal v-model:show="visible" preset="card" :title="$t('page.gateway.aiKey.batchTitle')" class="w-480px">
    <div class="flex-col-stretch gap-16px">
      <NAlert type="info" :show-icon="true">{{ $t('page.gateway.aiKey.batchTip') }}</NAlert>

      <template v-if="!result">
        <NRadioGroup v-model:value="mode" class="px-4px">
          <NSpace>
            <NRadio v-for="opt in modeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadio>
          </NSpace>
        </NRadioGroup>
        <DeptTreeSelect v-if="mode === 'dept'" v-model:value="deptId" class="w-full" />
        <NSelect
          v-else
          v-model:value="userIds"
          multiple
          filterable
          :options="userOptions"
          :placeholder="$t('page.gateway.aiKey.batchUsersRequired')"
        />
      </template>

      <template v-else>
        <div class="flex-center gap-16px py-8px text-14px">
          <span>{{ $t('page.gateway.aiKey.batchResultTotal', { total: result.total }) }}</span>
          <NTag type="success">{{ $t('page.gateway.aiKey.batchResultCreated', { created: result.created }) }}</NTag>
          <NTag type="default">{{ $t('page.gateway.aiKey.batchResultSkipped', { skipped: result.skipped }) }}</NTag>
          <NTag :type="result.failed.length > 0 ? 'error' : 'default'">
            {{ $t('page.gateway.aiKey.batchResultFailedCount', { failed: result.failed.length }) }}
          </NTag>
        </div>
        <template v-if="result.failed.length > 0">
          <NDivider class="!my-8px">{{ $t('page.gateway.aiKey.batchResultFailed') }}</NDivider>
          <div class="max-h-240px overflow-auto">
            <NDataTable
              size="small"
              :columns="[
                { key: 'name', title: $t('page.gateway.aiKey.batchResultUser') },
                { key: 'reason', title: $t('page.gateway.aiKey.batchResultReason') }
              ]"
              :data="result.failed"
              :row-key="(row: Api.Gateway.AiKeyBatchCreateResult['failed'][number]) => row.userId"
            />
          </div>
        </template>
      </template>
    </div>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="handleClose">{{ $t('common.close') }}</NButton>
        <NButton v-if="!result" type="primary" :loading="submitting" @click="handleSubmit">
          {{ $t('page.gateway.aiKey.batchSubmit') }}
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped></style>
