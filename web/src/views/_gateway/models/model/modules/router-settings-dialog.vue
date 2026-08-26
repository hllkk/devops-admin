<script setup lang="ts">
import { ref, watch } from 'vue';
import { fetchGetModelList, fetchGetRouterSettings, fetchUpdateRouterSettings } from '@/service/api/gateway';
import { useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { ROUTING_STRATEGY_OPTIONS } from '@/constants/business/gateway';
import SvgIcon from '@/components/custom/svg-icon.vue';

defineOptions({ name: 'RouterSettingsDialog' });

const visible = defineModel<boolean>('visible', { default: false });

const { formRef, validate, restoreValidation } = useNaiveForm();

function createDefault(): Api.Gateway.RouterSettings {
  return {
    routingStrategy: 'simple-shuffle',
    fallbacks: [],
    allowedFails: 3,
    cooldownTime: 60,
    numRetries: 2,
    timeout: 30,
    config: {}
  };
}

const formModel = ref<Api.Gateway.RouterSettings>(createDefault());
const submitting = ref(false);
/** 降级链可选项(现有模型的 modelKey 去重,源模型与降级模型共用此池) */
const modelOptions = ref<{ label: string; value: string }[]>([]);

const strategyOptions = ROUTING_STRATEGY_OPTIONS.map(o => ({ label: $t(o.label), value: o.value }));

async function loadModelOptions() {
  const { data, error } = await fetchGetModelList({
    pageNum: 1,
    pageSize: 1000,
    name: null,
    modelKey: null,
    category: null,
    isActive: null,
    isPublished: null,
    params: {}
  });
  if (!error && data) {
    const keys = data.rows.map(m => m.modelKey).filter((k): k is string => !!k);
    modelOptions.value = Array.from(new Set(keys)).map(k => ({ label: k, value: k }));
  }
}

async function loadSettings() {
  const { data, error } = await fetchGetRouterSettings();
  if (!error && data) {
    formModel.value = {
      routingStrategy: data.routingStrategy || 'simple-shuffle',
      fallbacks: Array.isArray(data.fallbacks) ? data.fallbacks : [],
      allowedFails: data.allowedFails ?? 3,
      cooldownTime: data.cooldownTime ?? 60,
      numRetries: data.numRetries ?? 2,
      timeout: data.timeout ?? 30,
      config: data.config ?? {}
    };
  }
}

function addFallback() {
  formModel.value.fallbacks.push({ model: '', fallbacks: [] });
}

function removeFallback(idx: number) {
  formModel.value.fallbacks.splice(idx, 1);
}

async function handleSubmit() {
  await validate();
  submitting.value = true;
  // 剔除空行(源模型与降级列表都需非空)
  const payload: Api.Gateway.RouterSettingsParams = {
    routingStrategy: formModel.value.routingStrategy,
    fallbacks: formModel.value.fallbacks.filter(f => !!f.model && f.fallbacks.length > 0),
    allowedFails: formModel.value.allowedFails,
    cooldownTime: formModel.value.cooldownTime,
    numRetries: formModel.value.numRetries,
    timeout: formModel.value.timeout,
    config: formModel.value.config
  };
  const { error } = await fetchUpdateRouterSettings(payload);
  submitting.value = false;
  if (error) return;
  window.$message?.success($t('common.updateSuccess'));
  visible.value = false;
}

watch(visible, async open => {
  if (!open) return;
  await Promise.all([loadModelOptions(), loadSettings()]);
  restoreValidation();
});
</script>

<template>
  <NModal
    v-model:show="visible"
    :title="$t('page.gateway.router.title')"
    preset="card"
    class="w-640px max-w-90%"
    :mask-closable="false"
    content-style="max-height: calc(100vh - 220px); overflow-y: auto;"
  >
    <NForm ref="formRef" :model="formModel" label-placement="left" :label-width="110">
      <NFormItem :label="$t('page.gateway.router.col.routingStrategy')">
        <NSelect
          v-model:value="formModel.routingStrategy"
          :options="strategyOptions"
          :placeholder="$t('page.gateway.router.form.strategyPlaceholder')"
        />
      </NFormItem>
      <p class="mb-12px text-12px text-slate-400">{{ $t('page.gateway.router.desc') }}</p>

      <NGrid responsive="screen" item-responsive :x-gap="16">
        <NFormItemGi span="24 s:12" :label="$t('page.gateway.router.col.allowedFails')">
          <NInputNumber
            v-model:value="formModel.allowedFails"
            :min="0"
            clearable
            :placeholder="$t('page.gateway.router.form.allowedFailsPlaceholder')"
            class="w-full"
          />
        </NFormItemGi>
        <NFormItemGi span="24 s:12" :label="$t('page.gateway.router.col.cooldownTime')">
          <NInputNumber
            v-model:value="formModel.cooldownTime"
            :min="0"
            clearable
            :placeholder="$t('page.gateway.router.form.cooldownTimePlaceholder')"
            class="w-full"
          />
        </NFormItemGi>
        <NFormItemGi span="24 s:12" :label="$t('page.gateway.router.col.numRetries')">
          <NInputNumber
            v-model:value="formModel.numRetries"
            :min="0"
            clearable
            :placeholder="$t('page.gateway.router.form.numRetriesPlaceholder')"
            class="w-full"
          />
        </NFormItemGi>
        <NFormItemGi span="24 s:12" :label="$t('page.gateway.router.col.timeout')">
          <NInputNumber
            v-model:value="formModel.timeout"
            :min="0"
            clearable
            :placeholder="$t('page.gateway.router.form.timeoutPlaceholder')"
            class="w-full"
          />
        </NFormItemGi>
      </NGrid>

      <div class="mb-8px mt-12px text-13px font-500 text-slate-500">{{ $t('page.gateway.router.col.fallbacks') }}</div>
      <p class="mb-8px text-12px text-slate-400">{{ $t('page.gateway.router.form.fallbacksTip') }}</p>
      <div v-for="(fb, idx) in formModel.fallbacks" :key="idx" class="mb-8px flex items-center gap-8px">
        <NSelect
          v-model:value="fb.model"
          filterable
          :options="modelOptions"
          :placeholder="$t('page.gateway.router.form.modelPlaceholder')"
          class="flex-1"
        />
        <NSelect
          v-model:value="fb.fallbacks"
          multiple
          filterable
          :options="modelOptions"
          :placeholder="$t('page.gateway.router.form.fallbackModelsPlaceholder')"
          class="flex-1"
        />
        <NButton quaternary type="error" size="small" @click="removeFallback(idx)">
          <SvgIcon icon="material-symbols:delete-outline" class="text-icon" />
        </NButton>
      </div>
      <NButton dashed size="small" class="mt-4px" @click="addFallback">
        + {{ $t('page.gateway.router.form.addFallback') }}
      </NButton>
    </NForm>
    <template #footer>
      <NSpace :size="16" justify="end">
        <NButton @click="visible = false">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped></style>
