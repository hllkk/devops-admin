<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { UploadCustomRequestOptions } from 'naive-ui';
import {
  fetchCreateSkill,
  fetchUpdateSkill,
  fetchUploadSkillPackage
} from '@/service/api/gateway';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({ name: 'SkillOperateDrawer' });

interface Props {
  operateType: NaiveUI.TableOperateType;
  rowData?: Api.Gateway.Skill | null;
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

const title = computed(() => (props.operateType === 'add' ? $t('page.gateway.skill.add') : $t('page.gateway.skill.edit')));

type Model = Api.Gateway.SkillOperateParams;

const model = ref<Model>(createDefaultModel());

function createDefaultModel(): Model {
  return {
    skillId: null,
    name: '',
    version: '',
    author: '',
    description: '',
    category: '',
    tags: [],
    iconUrl: '',
    documentationUrl: '',
    agentInstallPrompt: '',
    usageInstructions: '',
    isActive: true
  };
}

const rules: Record<string, App.Global.FormRule> = {
  name: createRequiredRule($t('page.gateway.skill.form.name.required')),
  version: {
    trigger: ['blur', 'change'],
    validator: (_rule: unknown, value: string | null) => {
      if (!value) return true; // 留空默认 1.0.0
      if (!/^\d{1,5}(\.\d{1,5}){0,2}$/.test(value)) {
        return new Error($t('page.gateway.skill.form.version.invalid'));
      }
      return true;
    }
  }
};

const uploading = ref(false);

/** 当前包信息(本地态展示；编辑态从 rowData 初始化，上传成功后本地更新——不改 prop) */
const packageInfo = ref<{ originName: string; size: number } | null>(null);

function formatSize(size: number): string {
  if (!size) return '0 B';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}

function initModel() {
  if (props.rowData) {
    const row = props.rowData;
    packageInfo.value = row.zipFilename ? { originName: row.zipOriginName, size: row.zipSize || 0 } : null;
    model.value = {
      skillId: row.skillId,
      name: row.name,
      version: row.version,
      author: row.author,
      description: row.description,
      category: row.category,
      tags: row.tags ? [...row.tags] : [],
      iconUrl: row.iconUrl,
      documentationUrl: row.documentationUrl,
      agentInstallPrompt: row.agentInstallPrompt,
      usageInstructions: row.usageInstructions,
      isActive: row.isActive
    };
  } else {
    model.value = createDefaultModel();
    packageInfo.value = null;
  }
}

watch(visible, val => {
  if (val) {
    initModel();
    restoreValidation();
  }
});

async function handleSubmit() {
  await validate();
  const isAdd = props.operateType === 'add';
  const { error } = isAdd ? await fetchCreateSkill(model.value) : await fetchUpdateSkill(model.value);
  if (error) return;
  window.$message?.success(
    isAdd ? $t('common.addSuccess') : $t('common.updateSuccess')
  );
  emit('submitted');
}

/** zip 上传(NUpload custom-request)：仅编辑态可传(需要 skillId) */
async function handleUpload({ file, onFinish, onError }: UploadCustomRequestOptions) {
  const raw = file.file;
  if (!raw || !props.rowData?.skillId) {
    onError();
    return;
  }
  uploading.value = true;
  const { error, data } = await fetchUploadSkillPackage(props.rowData.skillId, raw);
  uploading.value = false;
  if (error) {
    window.$message?.error(error.message);
    onError();
    return;
  }
  window.$message?.success($t('page.gateway.skill.upload.success'));
  // 本地回填包信息(不改 prop；父级列表经 submitted 后续刷新兜底)
  packageInfo.value = {
    originName: data?.zipOriginName ?? raw.name,
    size: data?.zipSize ?? raw.size
  };
  onFinish();
}

function handleUploadChange() {
  // 列表展示用 file-list 非受控,上传结果经 handleUpload 回填
}
</script>

<template>
  <NDrawer v-model:show="visible" display-directive="show" :width="560" class="max-w-90%">
    <NDrawerContent :title="title" :native-scrollbar="false" closable>
      <NForm ref="formRef" :model="model" :rules="rules" label-placement="left" :label-width="110">
        <NFormItem :label="$t('page.gateway.skill.col.name')" path="name">
          <NInput v-model:value="model.name" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.version')" path="version">
          <NInput v-model:value="model.version" :placeholder="$t('page.gateway.skill.form.version.placeholder')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.author')">
          <NInput v-model:value="model.author" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.category')">
          <NInput v-model:value="model.category" placeholder="general" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.tags')">
          <NDynamicTags
            :value="model.tags ?? []"
            :placeholder="$t('page.gateway.skill.form.tagsPlaceholder')"
            @update:value="(val: string[]) => (model.tags = val)"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.iconUrl')">
          <NInput v-model:value="model.iconUrl" placeholder="https://..." />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.documentationUrl')">
          <NInput v-model:value="model.documentationUrl" placeholder="https://..." />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.description')">
          <NInput v-model:value="model.description" type="textarea" :rows="2" :placeholder="$t('common.placeholderInput')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.agentInstallPrompt')">
          <NInput
            v-model:value="model.agentInstallPrompt"
            type="textarea"
            :rows="3"
            :placeholder="$t('page.gateway.skill.form.agentInstallPromptPlaceholder')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.usageInstructions')">
          <NInput
            v-model:value="model.usageInstructions"
            type="textarea"
            :rows="3"
            :placeholder="$t('page.gateway.skill.form.usageInstructionsPlaceholder')"
          />
        </NFormItem>

        <!-- zip 包上传(编辑态) -->
        <NFormItem v-if="operateType === 'edit'" :label="$t('page.gateway.skill.col.zipPackage')">
          <div class="flex w-full flex-col gap-8px">
            <div v-if="packageInfo" class="text-12px text-slate-400">
              {{ $t('page.gateway.skill.upload.current') }}：{{ packageInfo.originName }}（{{ formatSize(packageInfo.size) }}）
            </div>
            <NUpload
              accept=".zip"
              :max="1"
              :custom-request="handleUpload"
              :default-upload="true"
              @change="handleUploadChange"
            >
              <NButton size="small" :loading="uploading">
                {{ packageInfo ? $t('page.gateway.skill.upload.replace') : $t('page.gateway.skill.upload.upload') }}
              </NButton>
            </NUpload>
            <p class="text-12px text-slate-400">{{ $t('page.gateway.skill.upload.tip') }}</p>
          </div>
        </NFormItem>
        <p v-else class="mb-12px ml-110px text-12px text-slate-400">
          {{ $t('page.gateway.skill.upload.tip') }}
        </p>

        <NFormItem :label="$t('page.gateway.skill.col.isActive')">
          <NSwitch :value="!!model.isActive" @update:value="(val: boolean) => (model.isActive = val)" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end" :size="12">
          <NButton @click="visible = false">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped></style>
