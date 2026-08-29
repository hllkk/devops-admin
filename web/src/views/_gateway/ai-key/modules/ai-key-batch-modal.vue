<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { fetchGetUserSelect } from '@/service/api/system';
import {
  fetchBatchCreateMainKeys,
  fetchBatchCreateSceneKeys,
  fetchGetAllKeyScenarios,
  fetchGetAvailableMcps,
  fetchGetAvailableModels,
  fetchGetAvailableSkills
} from '@/service/api/gateway';
import { $t } from '@/locales';
import DeptTreeSelect from '@/components/custom/dept-tree-select.vue';
import {
  BUDGET_DURATION_OPTIONS,
  RATE_LIMIT_MODE_OPTIONS
} from '@/constants/business/gateway';

defineOptions({ name: 'AiKeyBatchModal' });

interface Props {
  /** 初始操作模式(默认批量开通主 Key) */
  initialMode?: 'main' | 'scene';
  /** 复制模板(主 Key 行)：打开时预填授权/预算/限流配置到场景 Key 表单 */
  templateRow?: Api.Gateway.AiKey | null;
}

const props = withDefaults(defineProps<Props>(), {
  initialMode: 'main',
  templateRow: null
});

interface Emits {
  (e: 'submitted'): void;
}

const emit = defineEmits<Emits>();

const visible = defineModel<boolean>('visible', {
  default: false
});

/** 操作模式：批量开通个人主 Key / 批量建个人场景 Key */
const actionMode = ref<'main' | 'scene'>('main');
/** 目标方式：按部门(取部门下全部用户) / 指定用户 */
const mode = ref<'dept' | 'users'>('dept');
const deptId = ref<CommonType.IdType | null>(null);
const userIds = ref<CommonType.IdType[]>([]);
const userOptions = ref<CommonType.Option<CommonType.IdType>[]>([]);
const submitting = ref(false);

// 场景 Key 表单(资源配置整体作模板套到每个目标用户)
const sceneForm = ref(createDefaultSceneForm());

function createDefaultSceneForm(): Api.Gateway.AiKeyBatchSceneCreateParams {
  return {
    deptId: null,
    userIds: [],
    nameTemplate: '{username}-',
    description: '',
    scenarioId: null,
    models: [],
    mcps: [],
    skills: [],
    budgetLimit: null,
    budgetHardLimit: false,
    budgetDuration: '30d',
    rateLimitMode: 'none',
    tpmLimit: null,
    rpmLimit: null,
    isActive: true,
    expiresAt: null
  };
}

/** 开通结果(null=未提交；部分失败语义：failed 空数组=全部成功) */
const result = ref<Api.Gateway.AiKeyBatchCreateResult | null>(null);

const modeOptions = computed(() => [
  { label: $t('page.gateway.aiKey.batchModeDept'), value: 'dept' },
  { label: $t('page.gateway.aiKey.batchModeUsers'), value: 'users' }
]);
const budgetDurationOptions = computed(() => BUDGET_DURATION_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));
const rateLimitModeOptions = computed(() => RATE_LIMIT_MODE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value })));

const modelOptions = ref<{ label: string; value: string }[]>([]);
const mcpOptions = ref<{ label: string; value: string }[]>([]);
const skillOptions = ref<{ label: string; value: string }[]>([]);
const scenarioOptions = ref<{ label: string; value: string }[]>([]);

async function loadUserOptions() {
  const { error, data } = await fetchGetUserSelect();
  if (!error && data) {
    userOptions.value = data.map(item => ({
      label: `${item.nickName} ( ${item.userName} )`,
      value: item.userId
    }));
  }
}

async function loadResourceOptions() {
  const [m, mc, sk, sc] = await Promise.all([
    fetchGetAvailableModels(),
    fetchGetAvailableMcps(),
    fetchGetAvailableSkills(),
    fetchGetAllKeyScenarios()
  ]);
  if (!m.error && m.data) modelOptions.value = m.data.map(x => ({ label: x.name, value: x.modelKey }));
  if (!mc.error && mc.data) {
    mcpOptions.value = mc.data.map(x => ({ label: `${x.name} (${x.serverName})`, value: x.serverName }));
  }
  if (!sk.error && sk.data) {
    skillOptions.value = sk.data.map(x => ({ label: `${x.name} (v${x.version})`, value: String(x.skillId) }));
  }
  if (!sc.error && sc.data) scenarioOptions.value = sc.data.map(x => ({ label: x.name, value: String(x.scenarioId) }));
}

const isScene = computed(() => actionMode.value === 'scene');

function resetForm() {
  actionMode.value = props.initialMode ?? 'main';
  mode.value = 'dept';
  deptId.value = null;
  userIds.value = [];
  sceneForm.value = createDefaultSceneForm();
  result.value = null;

  // 复制主 Key 模板：预填授权/预算/限流配置(不含归属与名称)
  if (props.templateRow) {
    const t = props.templateRow;
    sceneForm.value.models = t.models ? [...t.models] : [];
    sceneForm.value.mcps = t.mcps ? [...t.mcps] : [];
    sceneForm.value.skills = t.skills ? [...t.skills] : [];
    sceneForm.value.budgetLimit = t.budgetLimit;
    sceneForm.value.budgetHardLimit = t.budgetHardLimit;
    sceneForm.value.budgetDuration = t.budgetDuration ?? '30d';
    sceneForm.value.rateLimitMode = t.rateLimitMode ?? 'none';
    sceneForm.value.tpmLimit = t.tpmLimit;
    sceneForm.value.rpmLimit = t.rpmLimit;
    sceneForm.value.expiresAt = t.expiresAt;
  }
}

function handleClose() {
  // 已产生结果时，关闭即通知父组件刷新列表
  if (result.value) emit('submitted');
  visible.value = false;
}

function validateTargets(): boolean {
  if (mode.value === 'dept' && !deptId.value) {
    window.$message?.warning($t('page.gateway.aiKey.batchDeptRequired'));
    return false;
  }
  if (mode.value === 'users' && userIds.value.length === 0) {
    window.$message?.warning($t('page.gateway.aiKey.batchUsersRequired'));
    return false;
  }
  return true;
}

async function handleSubmit() {
  if (!validateTargets()) return;
  if (isScene.value && !sceneForm.value.nameTemplate) {
    window.$message?.warning($t('page.gateway.aiKey.batchScene.nameTemplateRequired'));
    return;
  }
  submitting.value = true;
  const targets = mode.value === 'dept' ? { deptId: deptId.value } : { userIds: userIds.value };
  const { error, data } = isScene.value
    ? await fetchBatchCreateSceneKeys({ ...sceneForm.value, ...targets })
    : await fetchBatchCreateMainKeys(targets);
  submitting.value = false;
  if (error) return;
  // 防御规范化：契约是 failed 空数组=全部成功，但旧后端/异常数据可能给 null，
  // 直接 .length 会让渲染链 TypeError 崩溃(弹窗冻结关不掉)
  result.value = { ...data, failed: data.failed ?? [] };
  window.$message?.success(
    isScene.value
      ? $t('page.gateway.aiKey.batchScene.result', {
          created: data.created,
          failed: result.value.failed.length
        })
      : $t('page.gateway.aiKey.batchResult', {
          created: data.created,
          skipped: data.skipped,
          failed: result.value.failed.length
        })
  );
}

watch(visible, val => {
  if (val) {
    resetForm();
    if (userOptions.value.length === 0) loadUserOptions();
    if (modelOptions.value.length === 0) loadResourceOptions();
  }
});
</script>

<template>
  <NModal
    v-model:show="visible"
    preset="card"
    :title="isScene ? $t('page.gateway.aiKey.batchScene.title') : $t('page.gateway.aiKey.batchTitle')"
    class="w-560px max-w-90%"
    content-style="max-height: calc(100vh - 180px); overflow-y: auto;"
  >
    <div class="flex-col-stretch gap-16px">
      <NAlert v-if="!result" type="info" :show-icon="true">
        {{ isScene ? $t('page.gateway.aiKey.batchScene.tip') : $t('page.gateway.aiKey.batchTip') }}
      </NAlert>

      <template v-if="!result">
        <NRadioGroup v-model:value="actionMode" class="px-4px" :disabled="!!templateRow">
          <NSpace>
            <NRadio value="main">{{ $t('page.gateway.aiKey.batchTitle') }}</NRadio>
            <NRadio value="scene">{{ $t('page.gateway.aiKey.batchScene.title') }}</NRadio>
          </NSpace>
        </NRadioGroup>

        <!-- 目标选择(两模式共用) -->
        <NRadioGroup v-model:value="mode" class="px-4px">
          <NSpace>
            <NRadio v-for="opt in modeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadio>
          </NSpace>
        </NRadioGroup>
        <DeptTreeSelect v-if="mode === 'dept'" v-model:value="deptId" class="w-full" />
        <NSelect
          v-else
          v-model:value="userIds"
          multiple
          filterable
          :options="userOptions"
          :placeholder="$t('page.gateway.aiKey.batchUsersRequired')"
        />

        <!-- 场景 Key 专属配置 -->
        <template v-if="isScene">
          <NFormItem :label="$t('page.gateway.aiKey.batchScene.nameTemplate')" :show-feedback="false">
            <NInput v-model:value="sceneForm.nameTemplate" :placeholder="$t('page.gateway.aiKey.batchScene.nameTemplatePlaceholder')" />
          </NFormItem>
          <p class="ml-2px text-12px text-slate-400">{{ $t('page.gateway.aiKey.batchScene.nameTemplateTip') }}</p>
          <NFormItem :label="$t('page.gateway.aiKey.col.scenario')" :show-feedback="false">
            <NSelect
              v-model:value="sceneForm.scenarioId"
              clearable
              filterable
              :options="scenarioOptions"
              :placeholder="$t('page.gateway.aiKey.form.scenarioPlaceholder')"
            />
          </NFormItem>
          <NFormItem :label="$t('page.gateway.aiKey.col.models')" :show-feedback="false">
            <NSelect v-model:value="sceneForm.models" multiple filterable :options="modelOptions" :placeholder="$t('page.gateway.aiKey.form.modelsPlaceholder')" />
          </NFormItem>
          <NFormItem :label="$t('page.gateway.aiKey.col.mcps')" :show-feedback="false">
            <NSelect v-model:value="sceneForm.mcps" multiple filterable :options="mcpOptions" :placeholder="$t('page.gateway.aiKey.form.mcpsPlaceholder')" />
          </NFormItem>
          <NFormItem :label="$t('page.gateway.aiKey.col.skills')" :show-feedback="false">
            <NSelect v-model:value="sceneForm.skills" multiple filterable :options="skillOptions" :placeholder="$t('page.gateway.aiKey.form.skillsPlaceholder')" />
          </NFormItem>
          <NFormItem :label="$t('page.gateway.aiKey.col.budgetLimit')" :show-feedback="false">
            <NInputNumber v-model:value="sceneForm.budgetLimit" :min="0" clearable :placeholder="$t('page.gateway.common.unlimited')" class="w-full">
              <template #suffix>¥</template>
            </NInputNumber>
          </NFormItem>
          <NFormItem :label="$t('page.gateway.aiKey.col.budgetDuration')" :show-feedback="false">
            <NSelect v-model:value="sceneForm.budgetDuration" :options="budgetDurationOptions" :placeholder="$t('common.placeholderSelect')" />
          </NFormItem>
          <NFormItem :label="$t('page.gateway.aiKey.col.budgetHardLimit')" :show-feedback="false">
            <NSpace align="center" :size="8" :wrap="false">
              <NSwitch :value="!!sceneForm.budgetHardLimit" @update:value="(v: boolean) => (sceneForm.budgetHardLimit = v)" />
              <span class="text-12px text-slate-400">{{ $t('page.gateway.aiKey.form.budgetHardLimitDesc') }}</span>
            </NSpace>
          </NFormItem>
          <NFormItem :label="$t('page.gateway.aiKey.col.rateLimitMode')" :show-feedback="false">
            <NSelect v-model:value="sceneForm.rateLimitMode" :options="rateLimitModeOptions" :placeholder="$t('common.placeholderSelect')" />
          </NFormItem>
          <NFormItem v-if="sceneForm.rateLimitMode === 'total'" :label="$t('page.gateway.aiKey.col.rateLimit')" :show-feedback="false">
            <div class="flex w-full gap-8px">
              <NInputNumber v-model:value="sceneForm.tpmLimit" :min="0" clearable placeholder="TPM" class="w-full" />
              <NInputNumber v-model:value="sceneForm.rpmLimit" :min="0" clearable placeholder="RPM" class="w-full" />
            </div>
          </NFormItem>
          <NFormItem :label="$t('page.gateway.aiKey.col.expiresAt')" :show-feedback="false">
            <NDatePicker
              v-model:formatted-value="sceneForm.expiresAt"
              type="datetime"
              value-format="yyyy-MM-dd'T'HH:mm:ssXXX"
              clearable
              :placeholder="$t('page.gateway.aiKey.form.expiresAtPlaceholder')"
              class="w-full"
            />
          </NFormItem>
        </template>
      </template>

      <template v-else>
        <div class="flex-center gap-16px py-8px text-14px">
          <span>{{ $t('page.gateway.aiKey.batchResultTotal', { total: result.total }) }}</span>
          <NTag type="success">{{ $t('page.gateway.aiKey.batchResultCreated', { created: result.created }) }}</NTag>
          <NTag v-if="!isScene" type="default">{{ $t('page.gateway.aiKey.batchResultSkipped', { skipped: result.skipped }) }}</NTag>
          <NTag :type="result.failed.length > 0 ? 'error' : 'default'">
            {{ $t('page.gateway.aiKey.batchResultFailedCount', { failed: result.failed.length }) }}
          </NTag>
        </div>
        <template v-if="result.failed.length > 0">
          <NDivider class="!my-8px">{{ $t('page.gateway.aiKey.batchResultFailed') }}</NDivider>
          <div class="max-h-240px overflow-auto">
            <NDataTable
              size="small"
              :columns="[
                { key: 'name', title: $t('page.gateway.aiKey.batchResultUser') },
                { key: 'reason', title: $t('page.gateway.aiKey.batchResultReason') }
              ]"
              :data="result.failed"
              :row-key="row => row.userId"
            />
          </div>
        </template>
      </template>
    </div>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="handleClose">{{ $t('common.close') }}</NButton>
        <NButton v-if="!result" type="primary" :loading="submitting" @click="handleSubmit">
          {{ isScene ? $t('page.gateway.aiKey.batchScene.submit') : $t('page.gateway.aiKey.batchSubmit') }}
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped></style>
