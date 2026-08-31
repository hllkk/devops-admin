<script setup lang="ts">
import { computed, onMounted, ref, toRaw } from 'vue';
import { jsonClone } from '@sa/utils';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { fetchGetDeptTree } from '@/service/api/system';
import UserSelect from '@/components/custom/user-select.vue';

defineOptions({ name: 'CostSearch' });

interface Emits {
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = defineModel<Api.Gateway.CostSearchParams>('model', { required: true });

const defaultModel = jsonClone(toRaw(model.value));

/** 预设时间档(业务日);custom=自定义区间(无按钮选中态),month=缺省 */
const presetKeys = ['today', 'yesterday', 'month', 'last7', 'last30'] as const;

type PresetKey = (typeof presetKeys)[number] | 'custom';

const preset = ref<PresetKey>('month');

function fmtDay(d: Date) {
  const y = d.getFullYear();
  const m = `${d.getMonth() + 1}`.padStart(2, '0');
  const day = `${d.getDate()}`.padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function applyPreset(key: PresetKey) {
  preset.value = key;
  const now = new Date();
  const end = fmtDay(now);
  let start = end;
  switch (key) {
    case 'today':
    case 'custom':
      break;
    case 'yesterday': {
      const y = new Date(now.getTime() - 86400000);
      start = fmtDay(y);
      model.value.startDate = start;
      model.value.endDate = start;
      return;
    }
    case 'month':
      start = `${now.getFullYear()}-${`${now.getMonth() + 1}`.padStart(2, '0')}-01`;
      break;
    case 'last7':
      start = fmtDay(new Date(now.getTime() - 6 * 86400000));
      break;
    case 'last30':
      start = fmtDay(new Date(now.getTime() - 29 * 86400000));
      break;
  }
  model.value.startDate = start;
  model.value.endDate = end;
}

/** 自定义起止(业务日);清空回退本月 */
const dateRange = ref<[number, number] | null>(null);

function syncDateRange() {
  if (dateRange.value) {
    model.value.startDate = fmtDay(new Date(dateRange.value[0]));
    model.value.endDate = fmtDay(new Date(dateRange.value[1]));
    preset.value = 'custom';
  } else {
    applyPreset('month');
  }
}

const presetOptions = computed(() =>
  presetKeys.map(key => ({
    label: $t(`page.gateway.cost.preset.${key}`),
    value: key
  }))
);

/** 部门树(单选,筛选含子树) */
const deptOptions = ref<Api.Common.CommonTreeRecord>([]);

async function loadDeptTree() {
  if (deptOptions.value.length > 0) return;
  const { data, error } = await fetchGetDeptTree();
  if (!error && data) deptOptions.value = data;
}

onMounted(() => {
  loadDeptTree();
  if (!model.value.startDate || !model.value.endDate) applyPreset('month');
});

async function reset() {
  await restoreValidation();
  dateRange.value = null;
  Object.assign(model.value, defaultModel);
  applyPreset('month');
  emit('search');
}

async function search() {
  await validate();
  if (dateRange.value) syncDateRange();
  emit('search');
}
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper">
    <NForm ref="formRef" :model="model" label-placement="left" :label-width="70" class="flex flex-col gap-12px">
      <div class="flex flex-wrap items-center gap-12px">
        <NRadioGroup :value="preset" size="small" @update:value="(v: string) => applyPreset(v as PresetKey)">
          <NRadioButton v-for="opt in presetOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadioButton>
        </NRadioGroup>
        <NDatePicker v-model:value="dateRange" type="daterange" clearable class="w-260px" @update:value="syncDateRange" />
      </div>
      <NGrid responsive="screen" item-responsive x-gap="12" y-gap="0">
        <NFormItemGi
          span="24 s:12 m:6"
          :label="$t('page.gateway.cost.search.department')"
          path="departmentId"
          class="pr-12px"
        >
          <NTreeSelect
            v-model:value="model.departmentId"
            clearable
            filterable
            key-field="id"
            label-field="label"
            :options="deptOptions as []"
            :placeholder="$t('common.placeholderSelect')"
          />
        </NFormItemGi>
        <NFormItemGi span="24 s:12 m:6" :label="$t('page.gateway.cost.search.user')" path="userId" class="pr-12px">
          <UserSelect
            v-model:value="model.userId"
            clearable
            filterable
            :placeholder="$t('page.gateway.usage.userPlaceholder')"
          />
        </NFormItemGi>
        <NFormItemGi span="24 s:12 m:6" :label="$t('page.gateway.cost.search.model')" path="model" class="pr-12px">
          <NInput
            v-model:value="model.model"
            clearable
            :placeholder="$t('page.gateway.cost.search.modelPlaceholder')"
            @keyup.enter="search"
          />
        </NFormItemGi>
        <NFormItemGi span="24 s:12 m:6" :label="$t('page.gateway.cost.search.provider')" path="provider" class="pr-12px">
          <NInput
            v-model:value="model.provider"
            clearable
            :placeholder="$t('page.gateway.usage.providerPlaceholder')"
            @keyup.enter="search"
          />
        </NFormItemGi>
      </NGrid>
      <NSpace justify="end">
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
    </NForm>
  </NCard>
</template>

<style scoped></style>
