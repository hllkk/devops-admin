<script setup lang="ts">
import { computed, onMounted, ref, toRaw } from 'vue';
import { jsonClone } from '@sa/utils';
import { fetchGetProviderList } from '@/service/api/gateway';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { ACTIVE_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'CredentialSearch' });

interface Emits {
  (e: 'reset'): void;
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = defineModel<Api.Gateway.CredentialSearchParams>('model', { required: true });

const defaultModel = jsonClone(toRaw(model.value));

const providerOptions = ref<{ label: string; value: CommonType.IdType }[]>([]);

async function loadProviders() {
  const { data, error } = await fetchGetProviderList({
    pageNum: 1,
    pageSize: 100,
    name: null,
    providerType: null,
    billingType: null,
    isActive: null,
    params: {}
  });
  if (!error && data) {
    providerOptions.value = data.rows.map(p => ({ label: p.name, value: p.providerId }));
  }
}

onMounted(loadProviders);

const activeOptions = computed(() => ACTIVE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const syncedOptions = computed(() => [
  { label: $t('page.gateway.common.synced'), value: 1 },
  { label: $t('page.gateway.common.unsynced'), value: 0 }
]);

const isActiveSearch = computed<number | null>({
  get: () => (model.value.isActive === null ? null : model.value.isActive ? 1 : 0),
  set: v => {
    model.value.isActive = v === null ? null : v === 1;
  }
});

const syncedSearch = computed<number | null>({
  get: () => (model.value.litellmSynced === null ? null : model.value.litellmSynced ? 1 : 0),
  set: v => {
    model.value.litellmSynced = v === null ? null : v === 1;
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
      <NCollapseItem :title="$t('common.search')" name="credential-search">
        <NForm ref="formRef" :model="model" label-placement="left" :label-width="80">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.credential.col.credentialName')" path="credentialName" class="pr-24px">
              <NInput v-model:value="model.credentialName" clearable :placeholder="$t('common.placeholderInput')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.credential.col.provider')" path="providerId" class="pr-24px">
              <NSelect v-model:value="model.providerId" clearable filterable :options="providerOptions" :placeholder="$t('common.placeholderSelect')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.credential.col.isActive')" path="isActive" class="pr-24px">
              <NSelect v-model:value="isActiveSearch" clearable :options="activeOptions" :placeholder="$t('common.placeholderSelect')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.credential.col.litellmSynced')" path="litellmSynced" class="pr-24px">
              <NSelect v-model:value="syncedSearch" clearable :options="syncedOptions" :placeholder="$t('common.placeholderSelect')" />
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
