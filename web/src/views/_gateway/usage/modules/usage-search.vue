<script setup lang="ts">
import { ref, toRaw } from 'vue';
import { jsonClone } from '@sa/utils';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import UserSelect from '@/components/custom/user-select.vue';

defineOptions({ name: 'UsageSearch' });

interface Emits {
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = defineModel<Api.Gateway.UsageLogSearchParams>('model', { required: true });

const defaultModel = jsonClone(toRaw(model.value));

/** 时间范围(ms 时间戳,空=不限)；提交转 RFC3339(后端按 UTC 解析) */
const dateRange = ref<[number, number] | null>(null);

function fmtRFC3339(ts: number) {
  return new Date(ts).toISOString();
}

function resetModel() {
  dateRange.value = null;
  Object.assign(model.value, defaultModel);
  // startTime/endTime 不在初始 model 里,assign 不会覆盖,需显式清空
  model.value.startTime = null;
  model.value.endTime = null;
}

async function reset() {
  await restoreValidation();
  resetModel();
  emit('search');
}

async function search() {
  await validate();
  model.value.startTime = dateRange.value ? fmtRFC3339(dateRange.value[0]) : null;
  model.value.endTime = dateRange.value ? fmtRFC3339(dateRange.value[1]) : null;
  emit('search');
}
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper">
    <NCollapse>
      <NCollapseItem :title="$t('common.search')" name="usage-search">
        <NForm ref="formRef" :model="model" label-placement="left" :label-width="80">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.usage.col.time')" path="startTime" class="pr-24px">
              <NDatePicker
                v-model:value="dateRange"
                type="datetimerange"
                clearable
                :default-time="['00:00:00', '23:59:59']"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.usage.col.user')" path="userId" class="pr-24px">
              <UserSelect
                v-model:value="model.userId"
                clearable
                filterable
                :placeholder="$t('page.gateway.usage.userPlaceholder')"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.usage.col.model')" path="model" class="pr-24px">
              <NInput
                v-model:value="model.model"
                clearable
                :placeholder="$t('page.gateway.usage.modelPlaceholder')"
                @keyup.enter="search"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.usage.provider')" path="provider" class="pr-24px">
              <NInput
                v-model:value="model.provider"
                clearable
                :placeholder="$t('page.gateway.usage.providerPlaceholder')"
                @keyup.enter="search"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:16" class="pr-24px">
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
