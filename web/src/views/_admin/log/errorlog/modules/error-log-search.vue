<script setup lang="ts">
import { ref, toRaw } from 'vue';
import { jsonClone } from '@sa/utils';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({
  name: 'ErrorLogSearch'
});

interface Emits {
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = defineModel<Api.Log.ErrorLogSearchParams>('model', { required: true });

const defaultModel = jsonClone(toRaw(model.value));

const dateRange = ref<[string, string] | null>(null);

function onDateRangeUpdate(value: [string, string] | null) {
  model.value.createdAtRange = value;
}

const levelOptions = [
  { label: '一般错误', value: 'error' },
  { label: '致命错误', value: 'fatal' }
];

const statusOptions = [
  { label: '未处理', value: '未处理' },
  { label: '处理中', value: '处理中' },
  { label: '处理完成', value: '处理完成' },
  { label: '处理失败', value: '处理失败' }
];

function resetModel() {
  dateRange.value = null;
  Object.assign(model.value, defaultModel);
}

async function reset() {
  await restoreValidation();
  resetModel();
  emit('search');
}

async function search() {
  await validate();
  emit('search');
}
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper">
    <NCollapse>
      <NCollapseItem :title="$t('common.search')" name="errorlog-search">
        <NForm ref="formRef" :model="model" label-placement="left" :label-width="80">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:6" :label="$t('page.log.errorlog.form')" path="form" class="pr-24px">
              <NInput v-model:value="model.form" :placeholder="$t('page.log.errorlog.placeholder.form')" clearable />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" :label="$t('page.log.errorlog.info')" path="info" class="pr-24px">
              <NInput v-model:value="model.info" :placeholder="$t('page.log.errorlog.placeholder.info')" clearable />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" :label="$t('page.log.errorlog.level')" path="level" class="pr-24px">
              <NSelect
                v-model:value="model.level"
                :placeholder="$t('page.log.errorlog.placeholder.level')"
                :options="levelOptions"
                clearable
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" :label="$t('page.log.errorlog.status')" path="status" class="pr-24px">
              <NSelect
                v-model:value="model.status"
                :placeholder="$t('page.log.errorlog.placeholder.status')"
                :options="statusOptions"
                clearable
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.log.errorlog.createTime')" path="createdAtRange" class="pr-24px">
              <NDatePicker
                v-model:formatted-value="dateRange"
                type="datetimerange"
                value-format="yyyy-MM-dd HH:mm:ss"
                clearable
                :default-time="['00:00:00', '23:59:59']"
                @update:formatted-value="onDateRangeUpdate"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" class="pr-24px">
              <NSpace class="w-full" justify="end">
                <NButton @click="reset">
                  <template #icon>
                    <icon-ic-round-refresh class="text-icon" />
                  </template>
                  {{ $t('common.reset') }}
                </NButton>
                <NButton type="primary" ghost @click="search">
                  <template #icon>
                    <icon-ic-round-search class="text-icon" />
                  </template>
                  {{ $t('common.search') }}
                </NButton>
              </NSpace>
            </NFormItemGi>
          </NGrid>
        </NForm>
      </NCollapseItem>
    </NCollapse>
  </NCard>
</template>

<style scoped></style>
