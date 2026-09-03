<script setup lang="ts">
import { computed, onMounted, ref, toRaw } from 'vue';
import { jsonClone } from '@sa/utils';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { fetchGetDeptTree } from '@/service/api/system';

defineOptions({ name: 'AdoptionSearch' });

interface Emits {
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const { formRef, validate, restoreValidation } = useNaiveForm();

const model = defineModel<Api.Gateway.AdoptionSearchParams>('model', { required: true });

const defaultModel = jsonClone(toRaw(model.value));

/** 预设时间档(业务日,复用成本分析语义);custom=自定义区间(无按钮选中态),month=缺省 */
const presetKeys = ['today', 'yesterday', 'month', 'last7', 'last30'] as const;

type PresetKey = (typeof presetKeys)[number] | 'custom';

const preset = ref<PresetKey>('month');

/** 自定义起止(业务日,与预设档互斥);清空回退本月 */
const dateRange = ref<[number, number] | null>(null);

function fmtDay(d: Date) {
  const y = d.getFullYear();
  const m = `${d.getMonth() + 1}`.padStart(2, '0');
  const day = `${d.getDate()}`.padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function applyPreset(key: PresetKey) {
  preset.value = key;
  dateRange.value = null;
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
    <NCollapse>
      <NCollapseItem :title="$t('common.search')" name="adoption-search">
        <NForm ref="formRef" :model="model" label-placement="left" :label-width="80">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.adoption.search.preset')" path="startDate" class="pr-24px">
              <NRadioGroup :value="preset" size="small" @update:value="(v: string) => applyPreset(v as PresetKey)">
                <NRadioButton v-for="opt in presetOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadioButton>
              </NRadioGroup>
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.adoption.search.dateRange')" path="endDate" class="pr-24px">
              <NDatePicker
                v-model:value="dateRange"
                type="daterange"
                clearable
                class="w-full"
                @update:value="syncDateRange"
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:8" :label="$t('page.gateway.adoption.search.department')" path="departmentId" class="pr-24px">
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
            <NFormItemGi span="24 m:24" class="pr-24px">
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
