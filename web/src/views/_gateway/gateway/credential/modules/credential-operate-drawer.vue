<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { jsonClone } from '@sa/utils';
import { fetchCreateCredential, fetchGetProviderList, fetchUpdateCredential } from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';
import { CREDENTIAL_FORMAT_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'CredentialOperateDrawer' });

interface Props {
  /** the type of operation */
  operateType: NaiveUI.TableOperateType;
  /** the edit row data */
  rowData?: Api.Gateway.Credential | null;
}

const props = defineProps<Props>();

interface Emits {
  (e: 'submitted'): void;
}

const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', {
  default: false
});

const { formRef, validate, restoreValidation } = useNaiveForm();
const { createRequiredRule } = useFormRules();

const title = computed(() => (props.operateType === 'add' ? $t('page.gateway.credential.add') : $t('page.gateway.credential.edit')));

type Model = Api.Gateway.CredentialOperateParams;

const model = ref<Model>(createDefaultModel());
const format = ref<Api.Gateway.CredentialFormat>('openai');
const valueEntries = ref<{ key: string; value: string }[]>([]);

function createDefaultModel(): Model {
  return {
    credentialId: null,
    credentialName: '',
    providerId: null,
    credentialValues: {},
    credentialInfo: { format: 'openai' },
    isActive: true,
    description: ''
  };
}

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

const formatOptions = computed(() => CREDENTIAL_FORMAT_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

const rules: Record<'credentialName' | 'providerId', App.Global.FormRule> = {
  credentialName: createRequiredRule($t('page.gateway.credential.form.credentialName.required')),
  providerId: createRequiredRule($t('page.gateway.credential.form.provider.required'))
};

/** 含 key/secret/token 的键按敏感值处理(密码框 + 眼睛切换) */
function isSecret(key: string) {
  return /(key|secret|token|password)/i.test(key);
}

function addEntry() {
  valueEntries.value.push({ key: '', value: '' });
}

function removeEntry(idx: number) {
  valueEntries.value.splice(idx, 1);
}

function handleUpdateModelWhenEdit() {
  model.value = createDefaultModel();
  format.value = 'openai';
  valueEntries.value = [
    { key: 'api_key', value: '' },
    { key: 'api_base', value: '' }
  ];

  if (props.operateType === 'edit' && props.rowData) {
    Object.assign(model.value, jsonClone(props.rowData));
    format.value = (props.rowData.credentialInfo?.format as Api.Gateway.CredentialFormat) ?? 'openai';
    const vals = props.rowData.credentialValues ?? {};
    const entries = Object.entries(vals).map(([k, v]) => ({ key: k, value: String(v ?? '') }));
    valueEntries.value = entries.length ? entries : [{ key: 'api_key', value: '' }];
  }
}

function closeDrawer() {
  visible.value = false;
}

async function handleSubmit() {
  await validate();

  // 组装 credentialValues：掩码值原样回传=后端保留旧明文，新值=覆盖
  const values: Record<string, string> = {};
  valueEntries.value.forEach(e => {
    if (e.key.trim()) values[e.key.trim()] = e.value;
  });
  model.value.credentialValues = values;
  model.value.credentialInfo = { format: format.value };

  if (props.operateType === 'add') {
    const { error } = await fetchCreateCredential(model.value);
    if (error) return;
    window.$message?.success($t('common.addSuccess'));
  }

  if (props.operateType === 'edit') {
    const { error } = await fetchUpdateCredential(model.value);
    if (error) return;
    window.$message?.success($t('common.updateSuccess'));
  }

  closeDrawer();
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
  <NDrawer v-model:show="visible" :title="title" display-directive="show" :width="640" class="max-w-90%">
    <NDrawerContent :title="title" :native-scrollbar="false" closable>
      <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="100">
        <NFormItem :label="$t('page.gateway.credential.col.provider')" path="providerId">
          <NSelect v-model:value="model.providerId" filterable :options="providerOptions" :placeholder="$t('common.placeholderSelect')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.credential.col.credentialName')" path="credentialName">
          <NInput v-model:value="model.credentialName" :disabled="operateType === 'edit'" :placeholder="$t('page.gateway.credential.form.credentialName.required')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.credential.col.format')" path="format">
          <NSelect v-model:value="format" :options="formatOptions" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.credential.col.credentialValues')">
          <div class="flex w-full flex-col gap-8px">
            <div v-for="(entry, idx) in valueEntries" :key="idx" class="flex items-center gap-8px">
              <NInput v-model:value="entry.key" :placeholder="$t('page.gateway.credential.form.keyPlaceholder')" class="w-160px shrink-0" />
              <NInput
                v-model:value="entry.value"
                :type="isSecret(entry.key) ? 'password' : 'text'"
                show-password-on="click"
                :placeholder="$t('page.gateway.credential.form.valuePlaceholder')"
                class="flex-1"
              />
              <NButton quaternary type="error" size="small" class="shrink-0" @click="removeEntry(idx)">
                <template #icon>
                  <icon-material-symbols-close-rounded class="text-icon" />
                </template>
              </NButton>
            </div>
            <NButton dashed size="small" @click="addEntry">
              {{ $t('page.gateway.credential.addKey') }}
            </NButton>
            <span class="text-12px text-slate-400">{{ $t('page.gateway.credential.form.valuesTip') }}</span>
          </div>
        </NFormItem>
        <NFormItem :label="$t('page.gateway.credential.col.isActive')" path="isActive">
          <NSwitch :value="!!model.isActive" @update:value="v => (model.isActive = v)" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.credential.col.description')" path="description">
          <NInput v-model:value="model.description" type="textarea" :rows="3" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace :size="16">
          <NButton @click="closeDrawer">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped></style>
