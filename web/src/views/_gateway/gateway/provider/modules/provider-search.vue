<script setup lang="ts">
import { computed, toRaw } from 'vue';
import { jsonClone } from '@sa/utils';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { ACTIVE_OPTIONS, BILLING_TYPE_OPTIONS, PROVIDER_TYPE_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'ProviderSearch' });

interface Emits {
  (e: 'reset'): void;
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = defineModel<Api.Gateway.ProviderSearchParams>('model', { required: true });

const defaultModel = jsonClone(toRaw(model.value));

const billingTypeOptions = computed(() => BILLING_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const activeOptions = computed(() => ACTIVE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

/** isActive 搜索用 1/0(number)，NSelect 不支持 boolean，此处转回 boolean */
const isActiveSearch = computed<number | null>({
  get: () => (model.value.isActive === null ? null : model.value.isActive ? 1 : 0),
  set: v => {
    model.value.isActive = v === null ? null : v === 1;
  }
});

function resetModel() {
  Object.assign(model.value, defaultModel);
}

async function reset() {
  await restoreValidation();
  resetModel();
  emit('reset');
}

async function search() {
  await validate();
  emit('search');
}
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper">
    <NCollapse>
      <NCollapseItem :title="$t('common.search')" name="provider-search">
        <NForm ref="formRef" :model="model" label-placement="left" :label-width="80">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.provider.col.name')" path="name" class="pr-24px">
              <NInput v-model:value="model.name" clearable :placeholder="$t('common.placeholderInput')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.provider.col.providerType')" path="providerType" class="pr-24px">
              <NSelect
                v-model:value="model.providerType"
                clearable
                filterable
                :options="PROVIDER_TYPE_OPTIONS"
                :placeholder="$t('common.placeholderSelect')"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.provider.col.billingType')" path="billingType" class="pr-24px">
              <NSelect v-model:value="model.billingType" clearable :options="billingTypeOptions" :placeholder="$t('common.placeholderSelect')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.provider.col.isActive')" path="isActive" class="pr-24px">
              <NSelect v-model:value="isActiveSearch" clearable :options="activeOptions" :placeholder="$t('common.placeholderSelect')" />
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
