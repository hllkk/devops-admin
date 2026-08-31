<script setup lang="ts">
import { computed, onMounted, ref, toRaw } from 'vue';
import type { SelectOption } from 'naive-ui';
import { jsonClone } from '@sa/utils';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { fetchGetMCPCategories } from '@/service/api/gateway';
import { ACTIVE_OPTIONS, MCP_HEALTH_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'McpSearch' });

interface Emits {
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = defineModel<Api.Gateway.MCPServerSearchParams>('model', { required: true });

const defaultModel = jsonClone(toRaw(model.value));

/** 分类受控选项(distinct)；拉取失败静默退化为可手输过滤 */
const categoryOptions = ref<SelectOption[]>([]);

onMounted(async () => {
  const { error, data } = await fetchGetMCPCategories();
  if (!error && data) {
    categoryOptions.value = data.map(c => ({ label: c, value: c }));
  }
});

/** 启停/发布状态用 1/0 选择(NSelect 不支持 boolean)，提交时转 boolean/null */
const activeValue = ref<0 | 1 | null>(null);
const publishedValue = ref<0 | 1 | null>(null);

const activeOptions = computed(() =>
  ACTIVE_OPTIONS.map(opt => ({ label: $t(opt.label), value: opt.value as number }))
);
const publishedOptions = computed(() => [
  { label: $t('page.gateway.common.published'), value: 1 },
  { label: $t('page.gateway.common.unpublished'), value: 0 }
]);
const healthOptions = computed(() =>
  MCP_HEALTH_OPTIONS.map(opt => ({ label: $t(opt.label), value: opt.value }))
);

function boolOf(v: 0 | 1 | null): boolean | null {
  if (v === 1) return true;
  if (v === 0) return false;
  return null;
}

async function reset() {
  await restoreValidation();
  Object.assign(model.value, defaultModel);
  activeValue.value = null;
  publishedValue.value = null;
  emit('search');
}

async function search() {
  await validate();
  model.value.isActive = boolOf(activeValue.value);
  model.value.isPublished = boolOf(publishedValue.value);
  emit('search');
}
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper">
    <NCollapse>
      <NCollapseItem :title="$t('common.search')" name="mcp-search">
        <NForm ref="formRef" :model="model" label-placement="left" :label-width="80">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.mcp.col.name')" path="name" class="pr-24px">
              <NInput
                v-model:value="model.name"
                clearable
                :placeholder="$t('common.keywordSearch')"
                @keyup.enter="search"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.mcp.col.category')" path="category" class="pr-24px">
              <NSelect
                v-model:value="model.category"
                clearable
                filterable
                tag
                :options="categoryOptions"
                :placeholder="$t('common.placeholderSelect')"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.mcp.col.isActive')" class="pr-24px">
              <NSelect v-model:value="activeValue" clearable :options="activeOptions" :placeholder="$t('common.placeholderSelect')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.mcp.col.isPublished')" class="pr-24px">
              <NSelect v-model:value="publishedValue" clearable :options="publishedOptions" :placeholder="$t('common.placeholderSelect')" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.mcp.col.healthStatus')" class="pr-24px">
              <NSelect v-model:value="model.healthStatus" clearable :options="healthOptions" :placeholder="$t('common.placeholderSelect')" />
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
