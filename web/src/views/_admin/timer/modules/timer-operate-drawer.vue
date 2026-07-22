<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { jsonClone } from '@sa/utils';
import { fetchCreateTimedTask, fetchUpdateTimedTask, fetchGetRegisteredMethods } from '@/service/api/system/timer';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({
  name: 'TimerOperateDrawer'
});

interface Props {
  operateType: NaiveUI.TableOperateType;
  rowData?: Api.System.SysTimedTask | null;
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

const title = computed(() => {
  const titles: Record<NaiveUI.TableOperateType, string> = {
    add: $t('page.system.timer.addTimer'),
    edit: $t('page.system.timer.editTimer')
  };
  return titles[props.operateType];
});

type Model = {
  ID: string;
  name: string;
  description: string;
  spec: string;
  withSeconds: boolean;
  executorType: 'method' | 'http';
  methodName: string;
  httpUrl: string;
  httpMethod: string;
  httpBody: string;
  httpAllowPrivate: boolean;
  enabled: boolean;
};

const model = ref<Model>(createDefaultModel());
const paramsText = ref('');
const headerText = ref('');
const methodOptions = ref<Api.System.RegisteredMethod[]>([]);

function createDefaultModel(): Model {
  return {
    ID: '0',
    name: '',
    description: '',
    spec: '',
    withSeconds: false,
    executorType: 'method',
    methodName: '',
    httpUrl: '',
    httpMethod: 'GET',
    httpBody: '',
    httpAllowPrivate: false,
    enabled: true
  };
}

type RuleKey = Extract<keyof Model, 'name' | 'spec' | 'methodName' | 'httpUrl'>;

const rules: Record<RuleKey, App.Global.FormRule> = {
  name: createRequiredRule($t('page.system.timer.form.name.required')),
  spec: createRequiredRule($t('page.system.timer.form.spec.required')),
  methodName: createRequiredRule($t('page.system.timer.form.methodName.required')),
  httpUrl: createRequiredRule($t('page.system.timer.form.httpUrl.required'))
};

const specPresets = [
  { label: '每天零点 @daily', value: '@daily' },
  { label: '每小时 @hourly', value: '@hourly' },
  { label: '每周 @weekly', value: '@weekly' },
  { label: '每30分钟', value: '*/30 * * * *' },
  { label: '每分钟', value: '* * * * *' },
  { label: '工作日9点', value: '0 9 * * 1-5' }
];

const httpMethodOptions = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'DELETE', value: 'DELETE' }
];

function handleUpdateModelWhenEdit() {
  model.value = createDefaultModel();
  paramsText.value = '';
  headerText.value = '';

  if (props.operateType === 'edit' && props.rowData) {
    const row = props.rowData;
    Object.assign(model.value, {
      ID: row.id,
      name: row.name,
      description: row.description || '',
      spec: row.spec,
      withSeconds: row.withSeconds,
      executorType: row.executorType,
      methodName: row.methodName || '',
      httpUrl: row.httpUrl || '',
      httpMethod: row.httpMethod || 'GET',
      httpBody: row.httpBody || '',
      httpAllowPrivate: row.httpAllowPrivate,
      enabled: row.enabled
    });
    paramsText.value = row.params ? JSON.stringify(row.params, null, 2) : '';
    headerText.value = row.httpHeader ? JSON.stringify(row.httpHeader, null, 2) : '';
  }
}

function closeDrawer() {
  visible.value = false;
}

function parseJsonField(text: string, label: string): { ok: boolean; value: Record<string, unknown> | null } {
  if (!text || !text.trim()) return { ok: true, value: null };
  try {
    return { ok: true, value: JSON.parse(text) };
  } catch {
    window.$message?.error(`${label} 不是合法 JSON`);
    return { ok: false, value: null };
  }
}

async function handleSubmit() {
  await validate();

  if (!model.value.name || !model.value.spec) {
    window.$message?.warning($t('page.system.timer.form.name.required'));
    return;
  }

  const p = parseJsonField(paramsText.value, $t('page.system.timer.params'));
  if (!p.ok) return;
  const h = parseJsonField(headerText.value, $t('page.system.timer.httpHeader'));
  if (!h.ok) return;

  const payload = {
    ...model.value,
    params: p.value,
    httpHeader: h.value
  };

  if (props.operateType === 'add') {
    const { error } = await fetchCreateTimedTask(payload);
    if (error) return;
    window.$message?.success($t('common.addSuccess'));
  }

  if (props.operateType === 'edit') {
    const { error } = await fetchUpdateTimedTask(payload);
    if (error) return;
    window.$message?.success($t('common.updateSuccess'));
  }

  closeDrawer();
  emit('submitted');
}

async function loadMethods() {
  if (methodOptions.value.length === 0) {
    const { data: result, error } = await fetchGetRegisteredMethods();
    if (!error && result) {
      methodOptions.value = result.methods || [];
    }
  }
}

watch(visible, () => {
  if (visible.value) {
    handleUpdateModelWhenEdit();
    restoreValidation();
    loadMethods();
  }
});
</script>

<template>
  <NDrawer v-model:show="visible" :title="title" display-directive="show" :width="800" class="max-w-90%">
    <NDrawerContent :title="title" :native-scrollbar="false" closable>
      <NForm ref="formRef" :model="model" :rules="rules">
        <NFormItem :label="$t('page.system.timer.name')" path="name" required>
          <NInput v-model:value="model.name" :placeholder="$t('page.system.timer.form.name.required')" />
        </NFormItem>
        <NFormItem :label="$t('page.system.timer.description')" path="description">
          <NInput v-model:value="model.description" type="textarea" :rows="2" placeholder="" />
        </NFormItem>
        <NFormItem :label="$t('page.system.timer.spec')" path="spec" required>
          <NSpace vertical class="w-full">
            <NSpace>
              <NInput v-model:value="model.spec" :placeholder="$t('page.system.timer.placeholder.spec')" style="flex: 1; min-width: 200px" />
              <NSelect
                :placeholder="$t('page.system.timer.specPreset')"
                :options="specPresets"
                style="width: 180px"
                @update:value="(v: string) => (model.spec = v)"
              />
            </NSpace>
            <span class="text-12px text-gray-400">{{ $t('page.system.timer.specHint') }}</span>
          </NSpace>
        </NFormItem>
        <NFormItem :label="$t('page.system.timer.withSeconds')" path="withSeconds">
          <NSpace align="center">
            <NSwitch v-model:value="model.withSeconds" />
            <span class="text-12px text-gray-400">{{ $t('page.system.timer.withSecondsHint') }}</span>
          </NSpace>
        </NFormItem>
        <NFormItem :label="$t('page.system.timer.executorType')" path="executorType" required>
          <NRadioGroup v-model:value="model.executorType">
            <NSpace>
              <NRadio value="method">{{ $t('page.system.timer.executorMethod') }}</NRadio>
              <NRadio value="http">{{ $t('page.system.timer.executorHttp') }}</NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>

        <template v-if="model.executorType === 'method'">
          <NFormItem :label="$t('page.system.timer.methodName')" path="methodName" required>
            <NSelect
              v-model:value="model.methodName"
              :options="methodOptions.map(m => ({ label: `${m.name} — ${m.description}`, value: m.name }))"
              :placeholder="$t('page.system.timer.placeholder.methodName')"
              filterable
            />
            <span class="text-12px text-gray-400 mt-4px">{{ $t('page.system.timer.methodNameHint') }}</span>
          </NFormItem>
          <NFormItem :label="$t('page.system.timer.params')" path="params">
            <NInput
              v-model:value="paramsText"
              type="textarea"
              :rows="4"
              :placeholder="$t('page.system.timer.paramsPlaceholder')"
            />
          </NFormItem>
        </template>

        <template v-if="model.executorType === 'http'">
          <NFormItem :label="$t('page.system.timer.httpUrl')" path="httpUrl" required>
            <NInput v-model:value="model.httpUrl" :placeholder="$t('page.system.timer.placeholder.httpUrl')" />
          </NFormItem>
          <NFormItem :label="$t('page.system.timer.httpMethod')" path="httpMethod">
            <NSelect v-model:value="model.httpMethod" :options="httpMethodOptions" style="width: 140px" />
          </NFormItem>
          <NFormItem :label="$t('page.system.timer.httpHeader')" path="httpHeader">
            <NInput
              v-model:value="headerText"
              type="textarea"
              :rows="2"
              :placeholder="$t('page.system.timer.placeholder.httpHeader')"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.timer.httpBody')" path="httpBody">
            <NInput v-model:value="model.httpBody" type="textarea" :rows="3" :placeholder="$t('page.system.timer.placeholder.httpBody')" />
          </NFormItem>
          <NFormItem :label="$t('page.system.timer.httpAllowPrivate')" path="httpAllowPrivate">
            <NSpace align="center">
              <NSwitch v-model:value="model.httpAllowPrivate" />
              <span class="text-12px text-gray-400">{{ $t('page.system.timer.httpAllowPrivateHint') }}</span>
            </NSpace>
          </NFormItem>
        </template>

        <NFormItem :label="$t('page.system.timer.enabled')" path="enabled">
          <NSwitch v-model:value="model.enabled" />
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
