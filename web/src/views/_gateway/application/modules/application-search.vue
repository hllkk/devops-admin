<script setup lang="ts">
import { toRaw } from 'vue';
import { jsonClone } from '@sa/utils';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import UserSelect from '@/components/custom/user-select.vue';

defineOptions({ name: 'ApplicationSearch' });

interface Emits {
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = defineModel<Api.Gateway.ApplicationSearchParams>('model', { required: true });

const defaultModel = jsonClone(toRaw(model.value));

/** 状态/类型选项(值空串=全部,后端空串忽略) */
const STATUS_OPTIONS = [
  { label: $t('page.gateway.application.statusAll'), value: '' },
  { label: $t('page.gateway.application.statusPending'), value: 'pending' },
  { label: $t('page.gateway.application.statusApproved'), value: 'approved' },
  { label: $t('page.gateway.application.statusRejected'), value: 'rejected' }
];
const TYPE_OPTIONS = [
  { label: $t('page.gateway.application.typeAll'), value: '' },
  { label: $t('page.gateway.application.typeModel'), value: 'model' },
  { label: $t('page.gateway.application.typeMcp'), value: 'mcp' },
  { label: $t('page.gateway.application.typeSkill'), value: 'skill' }
];

function resetModel() {
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
      <NCollapseItem :title="$t('common.search')" name="application-search">
        <NForm ref="formRef" :model="model" label-placement="left" :label-width="80">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.application.col.status')" path="status" class="pr-24px">
              <NSelect v-model:value="model.status" :options="STATUS_OPTIONS" />
            </NFormItemGi>
            <NFormItemGi
              span="24 s:12 m:8"
              :label="$t('page.gateway.application.col.resourceType')"
              path="resourceType"
              class="pr-24px"
            >
              <NSelect v-model:value="model.resourceType" :options="TYPE_OPTIONS" />
            </NFormItemGi>
            <NFormItemGi
              span="24 s:12 m:8"
              :label="$t('page.gateway.application.col.applicant')"
              path="userId"
              class="pr-24px"
            >
              <UserSelect
                v-model:value="model.userId"
                clearable
                filterable
                :placeholder="$t('page.gateway.application.userPlaceholder')"
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
