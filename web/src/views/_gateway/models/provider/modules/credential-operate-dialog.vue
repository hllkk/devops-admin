<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { fetchCreateCredential, fetchUpdateCredential } from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { CREDENTIAL_FORMAT_OPTIONS, FORMAT_API_BASE_PLACEHOLDER, FORMAT_NEEDS_NO_KEY } from '@/constants/business/gateway';

defineOptions({ name: 'CredentialOperateDialog' });

interface Props {
  operateType: NaiveUI.TableOperateType;
  rowData?: Api.Gateway.Credential | null;
  provider: Api.Gateway.Provider;
}
const props = defineProps<Props>();

interface Emits {
  (e: 'submitted'): void;
}
const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', { default: false });

const { formRef, validate, restoreValidation } = useNaiveForm();
const { createRequiredRule } = useFormRules();

const title = computed(() => (props.operateType === 'add' ? $t('page.gateway.credential.add') : $t('page.gateway.credential.edit')));

const credentialName = ref('');
const format = ref<Api.Gateway.CredentialFormat>('openai');
const apiBase = ref('');
const apiKey = ref('');
const isActive = ref(true);
const description = ref('');

const formModel = computed(() => ({
  credentialName: credentialName.value,
  format: format.value,
  apiBase: apiBase.value,
  apiKey: apiKey.value
}));

const formatOptions = computed(() => {
  const supported = (props.provider.supportedFormats ?? []) as string[];
  const opts = supported.length ? CREDENTIAL_FORMAT_OPTIONS.filter(o => supported.includes(o.value)) : CREDENTIAL_FORMAT_OPTIONS;
  return opts.map(o => ({ label: $t(o.label), value: o.value }));
});

/** Ollama 本地推理无鉴权，该类格式不展示 API Key 输入(对齐 AIHelms needsKey=false) */
const needsKey = computed(() => !FORMAT_NEEDS_NO_KEY.has(format.value));
const apiBasePlaceholder = computed(() => FORMAT_API_BASE_PLACEHOLDER[format.value] ?? 'https://api.example.com');

/** apiKey 仅在有鉴权格式下必填;编辑时掩码回显非空即可通过 */
const validateApiKey = (_rule: App.Global.FormRule, value: string) => {
  if (needsKey.value && !value) return new Error($t('page.gateway.credential.form.apiKey.required'));
  return true;
};

const rules: Record<'credentialName' | 'format' | 'apiBase' | 'apiKey', App.Global.FormRule> = {
  credentialName: createRequiredRule($t('page.gateway.credential.form.credentialName.required')),
  format: createRequiredRule($t('page.gateway.credential.col.format')),
  apiBase: createRequiredRule($t('page.gateway.credential.form.apiBase.required')),
  apiKey: { validator: validateApiKey }
};

function isMasked(v: string) {
  return v.includes('*');
}

function handleUpdateModelWhenEdit() {
  credentialName.value = '';
  format.value = ((props.provider.supportedFormats ?? [])[0] as Api.Gateway.CredentialFormat) ?? 'openai';
  apiBase.value = '';
  apiKey.value = '';
  isActive.value = true;
  description.value = '';

  if (props.operateType === 'edit' && props.rowData) {
    credentialName.value = props.rowData.credentialName;
    format.value = (props.rowData.credentialInfo?.format as Api.Gateway.CredentialFormat) ?? 'openai';
    apiBase.value = props.rowData.credentialValues?.api_base ?? '';
    apiKey.value = props.rowData.credentialValues?.api_key ?? '';
    isActive.value = props.rowData.isActive;
    description.value = props.rowData.description ?? '';
  }
}

function close() {
  visible.value = false;
}

async function handleSubmit() {
  await validate();

  // 组装 credentialValues: apiBase 明文总传; apiKey 掩码未改不传(后端保留旧), 新值才覆盖;
  // 无鉴权格式(如 ollama)不传 api_key
  const values: Record<string, string> = {};
  if (apiBase.value) values.api_base = apiBase.value;
  if (needsKey.value && apiKey.value && !isMasked(apiKey.value)) values.api_key = apiKey.value;

  const model: Api.Gateway.CredentialOperateParams = {
    credentialId: props.operateType === 'edit' ? props.rowData!.credentialId : null,
    credentialName: credentialName.value,
    providerId: props.provider.providerId,
    credentialValues: values,
    credentialInfo: { format: format.value },
    isActive: isActive.value,
    description: description.value
  };

  if (props.operateType === 'add') {
    const { error } = await fetchCreateCredential(model);
    if (error) return;
    window.$message?.success($t('common.addSuccess'));
  } else {
    const { error } = await fetchUpdateCredential(model);
    if (error) return;
    window.$message?.success($t('common.updateSuccess'));
  }
  close();
  emit('submitted');
}

watch(visible, () => {
  if (visible.value) {
    handleUpdateModelWhenEdit();
    restoreValidation();
  }
});
</script>

<template>
  <NModal v-model:show="visible" :title="title" preset="card" class="w-480px max-w-90%" :mask-closable="false" draggable>
    <NForm ref="formRef" :model="formModel" :rules="rules" label-placement="left" :label-width="90">
      <NFormItem :label="$t('page.gateway.credential.col.credentialName')" path="credentialName">
        <NInput v-model:value="credentialName" :disabled="operateType === 'edit'" :placeholder="$t('page.gateway.credential.form.credentialName.required')" />
      </NFormItem>
      <NFormItem :label="$t('page.gateway.credential.col.format')" path="format">
        <NSelect v-model:value="format" :options="formatOptions" :placeholder="$t('common.placeholderSelect')" />
      </NFormItem>
      <NFormItem :label="$t('page.gateway.credential.col.apiBase')" path="apiBase">
        <NInput v-model:value="apiBase" :placeholder="apiBasePlaceholder" />
      </NFormItem>
      <NFormItem v-if="needsKey" :label="$t('page.gateway.credential.col.apiKey')" path="apiKey">
        <NInput v-model:value="apiKey" type="password" show-password-on="click" :placeholder="$t('page.gateway.credential.form.apiKeyPlaceholder')" />
      </NFormItem>
      <NFormItem :label="$t('page.gateway.credential.col.isActive')">
        <NSwitch v-model:value="isActive" />
      </NFormItem>
    </NForm>
    <template #footer>
      <NSpace :size="16" justify="end">
        <NButton @click="close">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped></style>
