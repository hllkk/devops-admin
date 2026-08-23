<script setup lang="ts">
import { computed, toRaw } from 'vue';
import { jsonClone } from '@sa/utils';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { ACTIVE_OPTIONS, MODEL_CATEGORY_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'ModelSearch' });

interface Emits {
  (e: 'reset'): void;
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = defineModel<Api.Gateway.ModelSearchParams>('model', { required: true });

const defaultModel = jsonClone(toRaw(model.value));

const categoryOptions = computed(() => MODEL_CATEGORY_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const activeOptions = computed(() => ACTIVE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const publishedOptions = computed(() => [
  { label: $t('page.gateway.common.published'), value: 1 },
  { label: $t('page.gateway.common.unpublished'), value: 0 }
]);

const isActiveSearch = computed<number | null>({
  get: () => (model.value.isActive === null ? null : model.value.isActive ? 1 : 0),
  set: v => {
    model.value.isActive = v === null ? null : v === 1;
  }
});

const isPublishedSearch = computed<number | null>({
  get: () => (model.value.isPublished === null ? null : model.value.isPublished ? 1 : 0),
  set: v => {
    model.value.isPublished = v === null ? null : v === 1;
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
      <NCollapseItem :title="$t('common.search')" name="model-search">
        <NForm ref="formRef" :model="model" label-placement="left" :label-width="80">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.model.col.name')" path="name" class="pr-24px">
              <NInput v-model:value="model.name" clearable :placeholder="$t('common.placeholderInput')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.model.col.modelKey')" path="modelKey" class="pr-24px">
              <NInput v-model:value="model.modelKey" clearable :placeholder="$t('common.placeholderInput')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.model.col.category')" path="category" class="pr-24px">
              <NSelect v-model:value="model.category" clearable :options="categoryOptions" :placeholder="$t('common.placeholderSelect')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.model.col.isActive')" path="isActive" class="pr-24px">
              <NSelect v-model:value="isActiveSearch" clearable :options="activeOptions" :placeholder="$t('common.placeholderSelect')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.model.col.isPublished')" path="isPublished" class="pr-24px">
              <NSelect v-model:value="isPublishedSearch" clearable :options="publishedOptions" :placeholder="$t('common.placeholderSelect')" />
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
