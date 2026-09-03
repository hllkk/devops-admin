<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { SelectOption, UploadCustomRequestOptions, UploadFileInfo } from 'naive-ui';
import {
  fetchCreateSkill,
  fetchDownloadSkillPackage,
  fetchGetSkillCategories,
  fetchPublishSkill,
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

// ── 分类受控下拉(现有值可选,新值经 tag 输入,后端归一空值→general) ──

const categoryOptions = ref<SelectOption[]>([]);

/** NSelect 值(空串归一 null 以显示 placeholder；新值经 tag 输入回写 model) */
const categoryValue = computed<string | null>({
  get: () => model.value.category || null,
  set: val => {
    model.value.category = val ?? '';
  }
});

/** 拉取现有分类(distinct)做下拉选项;失败静默(不影响表单可用性,退化为手输) */
async function loadCategories() {
  const { error, data } = await fetchGetSkillCategories();
  if (!error && data) {
    categoryOptions.value = data.map(c => ({ label: c, value: c }));
  }
}

// ── 新建态 zip 暂存(提交时串联上传;建前无 skillId 不能即时上传) ──

const pendingFile = ref<File | null>(null);
const addFileList = ref<UploadFileInfo[]>([]);

function handleAddFileChange({ fileList }: { fileList: UploadFileInfo[] }) {
  pendingFile.value = fileList[0]?.file ?? null;
}

// ── 发布配置(合并进抽屉;发布到全员市场,不做部门/个人可见性) ──

const isPublish = ref(false);
const requireApproval = ref(false);
/** 回显初值：提交时对比，发布配置未变化则跳过 publish 请求(幂等但省一跳) */
let publishSnap = { isPublished: false, requiresApproval: false };

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
    isPublish.value = !!row.isPublished;
    requireApproval.value = !!row.requiresApproval;
  } else {
    model.value = createDefaultModel();
    packageInfo.value = null;
    pendingFile.value = null;
    addFileList.value = [];
    isPublish.value = false;
    requireApproval.value = false;
  }
  publishSnap = { isPublished: isPublish.value, requiresApproval: requireApproval.value };
}

// 未发布时审批勾选无意义，联动清掉
watch(isPublish, val => {
  if (!val) requireApproval.value = false;
});

watch(visible, val => {
  if (val) {
    initModel();
    restoreValidation();
    loadCategories();
  }
});

const submitting = ref(false);

async function handleSubmit() {
  await validate();
  if (submitting.value) return;
  // 勾了发布但无包：提前拦截(后端 PublishSkill 也会拦，这里反馈更快)
  if (isPublish.value) {
    const hasPackage = props.operateType === 'add' ? !!pendingFile.value : !!packageInfo.value;
    if (!hasPackage) {
      window.$message?.warning($t('page.gateway.skill.publish.needPackage'));
      return;
    }
  }
  submitting.value = true;
  try {
    if (props.operateType === 'add') {
      await handleAdd();
      return;
    }
    const { error } = await fetchUpdateSkill(model.value);
    if (error) return;
    if (!(await syncPublish(model.value.skillId))) {
      emit('submitted'); // 元数据已保存，刷新列表反映现状
      return;
    }
    window.$message?.success($t('common.updateSuccess'));
    emit('submitted');
  } finally {
    submitting.value = false;
  }
}

/** 新建：先建元数据，成功后串联上传暂存 zip(可选)与发布设置(可选)；
 * 后续步骤失败均保留已建记录并提示编辑重传/重试 */
async function handleAdd() {
  const { error, data } = await fetchCreateSkill(model.value);
  if (error) return;
  if (pendingFile.value && data?.skillId) {
    const up = await fetchUploadSkillPackage(data.skillId, pendingFile.value);
    if (up.error) {
      window.$message?.warning($t('page.gateway.skill.upload.createdButUploadFailed'));
      emit('submitted');
      return;
    }
  }
  if (!(await syncPublish(data?.skillId))) {
    window.$message?.warning($t('page.gateway.skill.publish.createdButPublishFailed'));
    emit('submitted');
    return;
  }
  window.$message?.success($t('common.addSuccess'));
  emit('submitted');
}

/** 发布配置变化时同步后端(可见性固定全员 all)；失败 warning 返回 false 由调用方收尾 */
async function syncPublish(skillId?: CommonType.IdType | null): Promise<boolean> {
  const id = skillId ?? model.value.skillId;
  if (!id) {
    return false; // 无 skillId 无法发布(建/改成功后必有，防御分支)
  }
  if (isPublish.value === publishSnap.isPublished && requireApproval.value === publishSnap.requiresApproval) {
    return true;
  }
  const { error } = await fetchPublishSkill({
    skillId: id,
    isPublished: isPublish.value,
    visibilityType: 'all',
    requiresApproval: requireApproval.value,
    departmentIds: [],
    userIds: []
  });
  if (error) {
    window.$message?.warning(error.message);
    return false;
  }
  return true;
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

const downloading = ref(false);

/** 下载当前包(管理端端点,不做用户侧发布/授权校验,不计次) */
async function handleDownload() {
  if (!props.rowData?.skillId) return;
  downloading.value = true;
  try {
    const blob = await fetchDownloadSkillPackage(props.rowData.skillId);
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = packageInfo.value?.originName || `${model.value.name}.zip`;
    a.click();
    URL.revokeObjectURL(url);
    window.$message?.success($t('page.gateway.skill.download.success'));
  } catch (err) {
    window.$message?.error(err instanceof Error ? err.message : $t('page.gateway.skill.download.failed'));
  } finally {
    downloading.value = false;
  }
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
          <NSelect
            v-model:value="categoryValue"
            :options="categoryOptions"
            :placeholder="$t('page.gateway.skill.form.categoryPlaceholder')"
            filterable
            tag
            clearable
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.tags')">
          <NDynamicTags
            :value="model.tags ?? []"
            :placeholder="$t('page.gateway.skill.form.tagsPlaceholder')"
            @update:value="(val: string[]) => (model.tags = val)"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.skill.col.iconUrl')">
          <IconPicker v-model:value="model.iconUrl" />
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

        <!-- zip 包：编辑态即时上传；新建态选择暂存、提交时串联上传 -->
        <NFormItem :label="$t('page.gateway.skill.col.zipPackage')">
          <div class="flex w-full flex-col gap-8px">
            <template v-if="operateType === 'edit'">
              <div v-if="packageInfo" class="flex items-center gap-8px text-12px text-slate-400">
                <span>
                  {{ $t('page.gateway.skill.upload.current') }}：{{ packageInfo.originName }}（{{ formatSize(packageInfo.size) }}）
                </span>
                <NButton size="tiny" :loading="downloading" @click="handleDownload">
                  {{ $t('page.gateway.skill.download.current') }}
                </NButton>
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
            </template>
            <NUpload
              v-else
              v-model:file-list="addFileList"
              accept=".zip"
              :max="1"
              :default-upload="false"
              @change="handleAddFileChange"
            >
              <NButton size="small">{{ $t('page.gateway.skill.upload.pick') }}</NButton>
            </NUpload>
            <p class="text-12px text-slate-400">{{ $t('page.gateway.skill.upload.tip') }}</p>
          </div>
        </NFormItem>

        <NFormItem :label="$t('page.gateway.skill.col.isActive')">
          <NSwitch :value="!!model.isActive" @update:value="(val: boolean) => (model.isActive = val)" />
        </NFormItem>

        <!-- 发布配置：发布到全员市场+领用审批；不做部门/个人可见性 -->
        <NFormItem :label="$t('page.gateway.skill.publish.short')">
          <div class="flex flex-col gap-4px">
            <NCheckbox v-model:checked="isPublish">
              {{ $t('page.gateway.skill.publish.toMarket') }}
            </NCheckbox>
            <NCheckbox v-model:checked="requireApproval" :disabled="!isPublish">
              {{ $t('page.gateway.skill.publish.requiresApproval') }}
            </NCheckbox>
          </div>
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end" :size="12">
          <NButton @click="visible = false">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="submitting" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped></style>
