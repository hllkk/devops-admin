<script setup lang="ts">
import { computed, ref } from 'vue';
import { $t } from '@/locales';
import { fetchGenerateReport } from '@/service/api/gateway';

defineOptions({ name: 'ReportGenerateModal' });

interface Emits {
  (e: 'generated', report: Api.Gateway.EfficiencyReportView): void;
}

const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', { required: true });

const reportType = ref<'weekly' | 'monthly' | 'custom'>('weekly');
const dateRange = ref<[number, number] | null>(null);
const generating = ref(false);

const typeOptions = computed(() =>
  (['weekly', 'monthly', 'custom'] as const).map(t => ({
    label: $t(`page.gateway.report.type.${t}`),
    value: t
  }))
);

function fmtDay(d: Date) {
  const y = d.getFullYear();
  const m = `${d.getMonth() + 1}`.padStart(2, '0');
  const day = `${d.getDate()}`.padStart(2, '0');
  return `${y}-${m}-${day}`;
}

async function handleGenerate() {
  const data: Api.Gateway.ReportGenerateParams = { reportType: reportType.value };
  if (reportType.value === 'custom') {
    if (!dateRange.value) {
      window.$message?.warning($t('page.gateway.report.generate.rangeRequired'));
      return;
    }
    data.startDate = fmtDay(new Date(dateRange.value[0]));
    data.endDate = fmtDay(new Date(dateRange.value[1]));
  }
  generating.value = true;
  const { data: view, error } = await fetchGenerateReport(data);
  generating.value = false;
  if (error || !view) return;
  window.$message?.success($t('page.gateway.report.generate.success'));
  visible.value = false;
  emit('generated', view);
}
</script>

<template>
  <NModal
    v-model:show="visible"
    preset="card"
    class="w-420px"
    :title="$t('page.gateway.report.generate.title')"
  >
    <NSpace vertical size="large">
      <NRadioGroup v-model:value="reportType" size="small">
        <NRadioButton v-for="opt in typeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadioButton>
      </NRadioGroup>
      <div v-if="reportType === 'weekly'" class="text-12px text-slate-400">
        {{ $t('page.gateway.report.generate.weeklyTip') }}
      </div>
      <div v-else-if="reportType === 'monthly'" class="text-12px text-slate-400">
        {{ $t('page.gateway.report.generate.monthlyTip') }}
      </div>
      <NDatePicker
        v-else
        v-model:value="dateRange"
        type="daterange"
        clearable
        class="w-full"
        :placeholder="$t('page.gateway.report.generate.dateRange')"
      />
      <NSpace justify="end">
        <NButton @click="visible = false">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="generating" @click="handleGenerate">
          {{ $t('page.gateway.report.generate.confirm') }}
        </NButton>
      </NSpace>
    </NSpace>
  </NModal>
</template>

<style scoped></style>
