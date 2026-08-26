<script setup lang="ts">
import { computed, toRaw, watch } from 'vue';
import { jsonClone } from '@sa/utils';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import UserSelect from '@/components/custom/user-select.vue';
import DeptTreeSelect from '@/components/custom/dept-tree-select.vue';
import { ACTIVE_OPTIONS, KEY_TYPE_OPTIONS, OWNER_TYPE_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'AiKeySearch' });

interface Emits {
  (e: 'reset'): void;
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = defineModel<Api.Gateway.AiKeySearchParams>('model', { required: true });

const defaultModel = jsonClone(toRaw(model.value));

const keyTypeOptions = computed(() => KEY_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const ownerTypeOptions = computed(() => OWNER_TYPE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const activeOptions = computed(() => ACTIVE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

// 切换归属类型时清空已选归属对象(避免 user 的 id 带到 dept)
watch(
  () => model.value.ownerType,
  () => {
    model.value.ownerId = null;
  }
);

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
      <NCollapseItem :title="$t('common.search')" name="ai-key-search">
        <NForm ref="formRef" :model="model" label-placement="left" :label-width="80">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.aiKey.col.name')" path="name" class="pr-24px">
              <NInput v-model:value="model.name" clearable :placeholder="$t('common.placeholderInput')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.aiKey.col.keyType')" path="keyType" class="pr-24px">
              <NSelect v-model:value="model.keyType" clearable :options="keyTypeOptions" :placeholder="$t('common.placeholderSelect')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.aiKey.col.ownerType')" path="ownerType" class="pr-24px">
              <NSelect v-model:value="model.ownerType" clearable :options="ownerTypeOptions" :placeholder="$t('common.placeholderSelect')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.aiKey.col.ownerId')" path="ownerId" class="pr-24px">
              <UserSelect
                v-if="model.ownerType === 'user'"
                v-model:value="model.ownerId"
                clearable
                :placeholder="$t('page.gateway.aiKey.form.ownerUserPlaceholder')"
                class="w-full"
              />
              <DeptTreeSelect
                v-else-if="model.ownerType === 'dept'"
                v-model:value="model.ownerId"
                clearable
                :placeholder="$t('page.gateway.aiKey.form.ownerDeptPlaceholder')"
                class="w-full"
              />
              <NSelect v-else disabled :placeholder="$t('page.gateway.aiKey.form.ownerType.required')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.aiKey.col.isActive')" path="isActive" class="pr-24px">
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
