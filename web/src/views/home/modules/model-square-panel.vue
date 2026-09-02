<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useClipboard } from '@vueuse/core';
import { $t } from '@/locales';
import {
  fetchDownloadSkill,
  fetchGetActiveMcps,
  fetchGetActiveModels,
  fetchGetActiveSkills,
  fetchGetMCPConnectConfig,
  fetchGetMyApplications,
  fetchSubmitApplication
} from '@/service/api/gateway';
import { MODEL_CATEGORY_OPTIONS, getProviderIcon } from '@/constants/business/gateway';
import SvgIcon from '@/components/custom/svg-icon.vue';

defineOptions({ name: 'HomeModelSquarePanel' });

/**
 * home「模型广场」Tab 面板：可见模型/MCP 浏览 + 接入信息 + 申请订阅。
 * identity 由 home 主页统一加载后传入（与 AI 身份卡/我的资源共享同一份数据，避免重复请求）。
 */
const props = defineProps<{
  identity: Api.Gateway.MyIdentity | null;
}>();

const emit = defineEmits<{
  applied: [];
}>();

const { copy: copyText } = useClipboard({ legacy: true, copiedDuring: 2000 });

const isLoading = ref(true);
const models = ref<Api.Gateway.ActiveModel[]>([]);
const mcps = ref<Api.Gateway.AvailableMcp[]>([]);
const skills = ref<Api.Gateway.AvailableSkill[]>([]);
const keyword = ref('');

/** 最新上架模型(接口按 model_id 倒序,首个即最新;再有新模型上架时 New 标识随之转移) */
const newestModelId = computed(() => models.value[0]?.modelId);

/** 广场资源类型切换(模型/MCP/Skill) */
const resourceKind = ref<'model' | 'mcp' | 'skill'>('model');

/** 主 Key 已授权 modelKey/serverName/skillId 集合(未开通为空集) */
const authorizedKeys = computed(() => new Set(props.identity?.opened ? props.identity.models : []));
const authorizedMcpNames = computed(() => new Set(props.identity?.opened ? props.identity.mcps : []));
const authorizedSkillIds = computed(() => new Set(props.identity?.opened ? props.identity.skills : []));

/** 已有待审批申请的资源键集合(`${type}:${id}`,本地维护 + 加载时从我的申请回填) */
const appliedKeys = ref<Set<string>>(new Set());

const filteredModels = computed(() => {
  const kw = keyword.value.trim().toLowerCase();
  if (!kw) return models.value;
  return models.value.filter(
    m => m.name.toLowerCase().includes(kw) || m.modelKey.toLowerCase().includes(kw)
  );
});

const filteredMcps = computed(() => {
  const kw = keyword.value.trim().toLowerCase();
  if (!kw) return mcps.value;
  return mcps.value.filter(
    m => m.name.toLowerCase().includes(kw) || m.serverName.toLowerCase().includes(kw)
  );
});

const filteredSkills = computed(() => {
  const kw = keyword.value.trim().toLowerCase();
  if (!kw) return skills.value;
  return skills.value.filter(
    s => s.name.toLowerCase().includes(kw) || s.author.toLowerCase().includes(kw)
  );
});

function isAuthorized(model: Api.Gateway.ActiveModel) {
  return authorizedKeys.value.has(model.modelKey);
}

function isMcpAuthorized(mcp: Api.Gateway.AvailableMcp) {
  return authorizedMcpNames.value.has(mcp.serverName);
}

function isSkillAuthorized(skill: Api.Gateway.AvailableSkill) {
  return authorizedSkillIds.value.has(String(skill.skillId));
}

function isApplied(model: Api.Gateway.ActiveModel) {
  return appliedKeys.value.has(`model:${model.modelId}`);
}

function isMcpApplied(mcp: Api.Gateway.AvailableMcp) {
  return appliedKeys.value.has(`mcp:${mcp.mcpServerId}`);
}

function isSkillApplied(skill: Api.Gateway.AvailableSkill) {
  return appliedKeys.value.has(`skill:${skill.skillId}`);
}

function categoryLabel(category: string) {
  const hit = MODEL_CATEGORY_OPTIONS.find(o => o.value === category);
  return hit ? $t(hit.label) : category;
}

// ===== 接入信息弹窗 =====
const accessVisible = ref(false);
const accessModel = ref<Api.Gateway.ActiveModel | null>(null);
const accessMcp = ref<Api.Gateway.MCPConnectConfig | null>(null);
const showFullKey = ref(false);

const fullKey = computed(() => (props.identity?.opened ? props.identity.keyValue : ''));
const maskedKey = computed(() => {
  const value = fullKey.value;
  if (!value) return '';
  return value.length > 12 ? `${value.slice(0, 7)}****${value.slice(-4)}` : value;
});
const displayKey = computed(() => (showFullKey.value ? fullKey.value : maskedKey.value));

function handleViewAccess(model: Api.Gateway.ActiveModel) {
  accessModel.value = model;
  accessMcp.value = null;
  showFullKey.value = false;
  accessVisible.value = true;
}

/** MCP 接入信息(后端组装客户端配置 JSON,含主 Key 鉴权头;页面只展示掩码 Key,配置经复制导出) */
async function handleViewMcpAccess(mcp: Api.Gateway.AvailableMcp) {
  accessModel.value = null;
  accessMcp.value = null;
  accessVisible.value = true;
  const { data } = await fetchGetMCPConnectConfig(mcp.mcpServerId);
  if (data) accessMcp.value = data;
}

function handleCopyConfig() {
  if (accessMcp.value?.config) handleCopy('mcpConfig', JSON.stringify(accessMcp.value.config, null, 2));
}

// ===== 申请订阅(需审批资源;P2 资源申请审批) =====
const applyVisible = ref(false);
const applyModel = ref<Api.Gateway.ActiveModel | null>(null);
const applyMcp = ref<Api.Gateway.AvailableMcp | null>(null);
const applySkill = ref<Api.Gateway.AvailableSkill | null>(null);
const applyReason = ref('');
const applySubmitting = ref(false);

function handleApply(model: Api.Gateway.ActiveModel) {
  applyModel.value = model;
  applyMcp.value = null;
  applySkill.value = null;
  applyReason.value = '';
  applyVisible.value = true;
}

function handleApplyMcp(mcp: Api.Gateway.AvailableMcp) {
  applyMcp.value = mcp;
  applyModel.value = null;
  applySkill.value = null;
  applyReason.value = '';
  applyVisible.value = true;
}

function handleApplySkill(skill: Api.Gateway.AvailableSkill) {
  applySkill.value = skill;
  applyModel.value = null;
  applyMcp.value = null;
  applyReason.value = '';
  applyVisible.value = true;
}

async function handleSubmitApply() {
  const reason = applyReason.value.trim();
  if ((!applyModel.value && !applyMcp.value && !applySkill.value) || !reason) return;
  applySubmitting.value = true;
  const resourceType = applyModel.value ? 'model' : applyMcp.value ? 'mcp' : 'skill';
  const resourceId = applyModel.value
    ? applyModel.value.modelId
    : applyMcp.value
      ? applyMcp.value.mcpServerId
      : applySkill.value!.skillId;
  const { error } = await fetchSubmitApplication({ resourceType, resourceId, reason });
  applySubmitting.value = false;
  if (error) return;
  window.$message?.success($t('page.home.square.applySuccess'));
  appliedKeys.value.add(`${resourceType}:${resourceId}`);
  applyVisible.value = false;
  // 通知 home 刷新「我的申请」列表(新 pending 记录)
  emit('applied');
}

// ===== Skill 下载(blob 触发保存;需审批 Skill 未授权时后端拒绝,提示先申请) =====
const downloadingSkillId = ref<string | number | null>(null);

async function handleDownloadSkill(skill: Api.Gateway.AvailableSkill) {
  if (!skill.hasPackage) {
    window.$message?.warning($t('page.home.square.skillNoPackage'));
    return;
  }
  if (skill.requiresApproval && !isSkillAuthorized(skill)) {
    window.$message?.warning($t('page.home.square.skillNeedApproval'));
    return;
  }
  downloadingSkillId.value = skill.skillId;
  try {
    const blob = await fetchDownloadSkill(skill.skillId);
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${skill.name}-v${skill.version}.zip`;
    a.click();
    URL.revokeObjectURL(url);
    window.$message?.success($t('page.home.square.skillDownloadSuccess'));
  } catch (err) {
    window.$message?.error(err instanceof Error ? err.message : $t('page.home.square.skillDownload'));
  } finally {
    downloadingSkillId.value = null;
  }
}

// ===== 接入弹窗复制(按字段隔离已复制标记,避免一处复制全部按钮亮"已复制") =====
const copiedField = ref<string | null>(null);
let copiedTimer: ReturnType<typeof setTimeout> | null = null;

function isCopied(field: string) {
  return copiedField.value === field;
}

async function handleCopy(field: string, value: string) {
  if (!value) return;
  try {
    await copyText(value);
    copiedField.value = field;
    if (copiedTimer) clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copiedField.value = null;
    }, 2000);
  } catch {
    window.$message?.error($t('common.copyNotSupported'));
  }
}

onMounted(async () => {
  const [activeRes, mcpRes, skillRes, appRes] = await Promise.all([
    fetchGetActiveModels(),
    fetchGetActiveMcps(),
    fetchGetActiveSkills(),
    fetchGetMyApplications({ status: 'pending', pageNum: 1, pageSize: 100, params: {} })
  ]);
  if (!activeRes.error && activeRes.data) models.value = activeRes.data;
  if (!mcpRes.error && mcpRes.data) mcps.value = mcpRes.data;
  if (!skillRes.error && skillRes.data) skills.value = skillRes.data;
  if (!appRes.error && appRes.data?.rows) {
    // 键含资源类型(model/mcp/skill),避免 resourceId 撞车
    appliedKeys.value = new Set(appRes.data.rows.map(r => `${r.resourceType}:${r.resourceId}`));
  }
  isLoading.value = false;
});
</script>

<template>
  <div class="flex flex-col gap-12px">
    <!-- 头部：标题/说明 + 搜索 -->
    <div class="flex flex-wrap items-center gap-12px px-4px">
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-8px">
          <SvgIcon icon="lucide:store" class="home-accent text-20px" />
          <span class="text-16px font-bold text-slate-900 dark:text-slate-100">
            {{ $t('page.home.square.title') }}
          </span>
        </div>
        <div class="mt-2px text-12px text-slate-400">{{ $t('page.home.square.subtitle') }}</div>
      </div>
      <NInput
        v-model:value="keyword"
        :placeholder="resourceKind === 'skill' ? $t('page.home.square.searchPlaceholderSkill') : $t('page.home.square.searchPlaceholder')"
        clearable
        class="w-220px rounded-12px"
      >
        <template #prefix>
          <SvgIcon icon="lucide:search" class="text-14px text-slate-400" />
        </template>
      </NInput>
    </div>

    <!-- 资源类型切换(模型/MCP/Skill) -->
    <div class="flex items-center gap-8px px-4px">
      <NRadioGroup v-model:value="resourceKind" size="small">
        <NRadioButton value="model">{{ $t('page.home.square.filterModels') }}</NRadioButton>
        <NRadioButton value="mcp">{{ $t('page.home.square.filterMcps') }}</NRadioButton>
        <NRadioButton value="skill">{{ $t('page.home.square.filterSkills') }}</NRadioButton>
      </NRadioGroup>
    </div>

    <NAlert v-if="identity && !identity.opened" type="warning" :show-icon="true">
      {{ $t('page.home.square.noIdentity') }}
    </NAlert>

    <!-- 模型卡片网格 -->
    <template v-if="resourceKind === 'model'">
      <div v-if="isLoading" class="h-240px w-full animate-pulse rounded-16px bg-slate-200/60 dark:bg-slate-700/40" />
      <div
        v-else-if="filteredModels.length"
        class="grid grid-cols-1 gap-12px sm:grid-cols-2 lg:grid-cols-3"
      >
        <NCard
          v-for="model in filteredModels"
          :key="model.modelId"
          :bordered="false"
          size="small"
          class="card-wrapper shadow-sm transition-shadow hover:shadow-md"
        >
          <div class="flex h-full flex-col gap-8px">
            <!-- 头部：logo + 名称 + 已授权标记 -->
            <div class="flex items-start gap-10px">
              <div class="flex size-40px shrink-0 items-center justify-center rounded-12px bg-slate-100 dark:bg-slate-700/60">
                <SvgIcon :local-icon="getProviderIcon(model.logoProviderType)" class="h-24px w-24px" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-4px">
                  <span class="truncate text-14px font-semibold">{{ model.name }}</span>
                  <NTag
                    v-if="model.modelId === newestModelId"
                    size="tiny"
                    type="primary"
                    :bordered="false"
                    class="shrink-0"
                  >
                    {{ $t('page.home.square.isNew') }}
                  </NTag>
                  <SvgIcon
                    v-if="isAuthorized(model)"
                    icon="lucide:circle-check"
                    class="shrink-0 text-16px text-emerald-500"
                  />
                </div>
                <code class="block truncate text-12px text-slate-400">{{ model.modelKey }}</code>
              </div>
            </div>
            <!-- 类别 + 能力标签 -->
            <div class="flex flex-wrap gap-4px">
              <NTag size="tiny" :bordered="false" type="primary">{{ categoryLabel(model.category) }}</NTag>
              <NTag v-for="cap in model.capabilities.slice(0, 3)" :key="cap" size="tiny" :bordered="false">
                {{ cap }}
              </NTag>
            </div>
            <!-- 描述(两行截断) -->
            <p class="line-clamp-2 min-h-32px text-12px text-slate-400">{{ model.description }}</p>
            <!-- 状态 + 操作按钮 -->
            <div class="mt-auto flex items-center justify-between gap-8px">
              <NTag v-if="isAuthorized(model)" type="success" size="small" :bordered="false">
                {{ $t('page.home.square.authorized') }}
              </NTag>
              <NTag v-else-if="isApplied(model)" type="warning" size="small" :bordered="false">
                {{ $t('page.home.square.applyPending') }}
              </NTag>
              <NTag v-else-if="model.requiresApproval" type="warning" size="small" :bordered="false">
                {{ $t('page.home.square.requiresApproval') }}
              </NTag>
              <NTag v-else size="small">{{ $t('page.home.square.notAuthorized') }}</NTag>
              <div class="flex items-center gap-6px">
                <NButton
                  v-if="model.requiresApproval && !isAuthorized(model) && !isApplied(model)"
                  size="tiny"
                  type="primary"
                  :disabled="!identity?.opened"
                  @click="handleApply(model)"
                >
                  {{ $t('page.home.square.apply') }}
                </NButton>
                <NButton
                  size="tiny"
                  type="primary"
                  ghost
                  :disabled="!identity?.opened"
                  @click="handleViewAccess(model)"
                >
                  {{ $t('page.home.square.viewAccess') }}
                </NButton>
              </div>
            </div>
          </div>
        </NCard>
      </div>
      <NEmpty v-else class="h-240px justify-center" :description="$t('page.home.square.empty')" />
    </template>

    <!-- MCP 卡片网格 -->
    <template v-else-if="resourceKind === 'mcp'">
      <div v-if="isLoading" class="h-240px w-full animate-pulse rounded-16px bg-slate-200/60 dark:bg-slate-700/40" />
      <div v-else-if="filteredMcps.length" class="grid grid-cols-1 gap-12px sm:grid-cols-2 lg:grid-cols-3">
        <NCard
          v-for="mcp in filteredMcps"
          :key="mcp.mcpServerId"
          :bordered="false"
          size="small"
          class="card-wrapper shadow-sm transition-shadow hover:shadow-md"
        >
          <div class="flex h-full flex-col gap-8px">
            <div class="flex items-start gap-10px">
              <div class="flex size-40px shrink-0 items-center justify-center rounded-12px bg-emerald-50 dark:bg-emerald-900/30">
                <SvgIcon icon="lucide:server" class="text-20px text-emerald-600" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-4px">
                  <span class="truncate text-14px font-semibold">{{ mcp.name }}</span>
                  <SvgIcon
                    v-if="isMcpAuthorized(mcp)"
                    icon="lucide:circle-check"
                    class="shrink-0 text-16px text-emerald-500"
                  />
                </div>
                <code class="block truncate text-12px text-slate-400">{{ mcp.serverName }}</code>
              </div>
            </div>
            <div class="flex flex-wrap gap-4px">
              <NTag size="tiny" :bordered="false" type="primary">{{ mcp.category }}</NTag>
              <NTag size="tiny" :bordered="false">
                {{ $t('page.home.square.toolsCount', { count: mcp.toolCount }) }}
              </NTag>
            </div>
            <p class="line-clamp-2 min-h-32px text-12px text-slate-400">{{ mcp.description }}</p>
            <div class="mt-auto flex items-center justify-between gap-8px">
              <NTag v-if="isMcpAuthorized(mcp)" type="success" size="small" :bordered="false">
                {{ $t('page.home.square.authorized') }}
              </NTag>
              <NTag v-else-if="isMcpApplied(mcp)" type="warning" size="small" :bordered="false">
                {{ $t('page.home.square.applyPending') }}
              </NTag>
              <NTag v-else-if="mcp.requiresApproval" type="warning" size="small" :bordered="false">
                {{ $t('page.home.square.requiresApproval') }}
              </NTag>
              <NTag v-else size="small">{{ $t('page.home.square.notAuthorized') }}</NTag>
              <div class="flex items-center gap-6px">
                <NButton
                  v-if="mcp.requiresApproval && !isMcpAuthorized(mcp) && !isMcpApplied(mcp)"
                  size="tiny"
                  type="primary"
                  :disabled="!identity?.opened"
                  @click="handleApplyMcp(mcp)"
                >
                  {{ $t('page.home.square.apply') }}
                </NButton>
                <NButton size="tiny" type="primary" ghost @click="handleViewMcpAccess(mcp)">
                  {{ $t('page.home.square.viewAccess') }}
                </NButton>
              </div>
            </div>
          </div>
        </NCard>
      </div>
      <NEmpty v-else class="h-240px justify-center" :description="$t('page.home.square.empty')" />
    </template>

    <!-- Skill 卡片网格 -->
    <template v-else>
      <div v-if="isLoading" class="h-240px w-full animate-pulse rounded-16px bg-slate-200/60 dark:bg-slate-700/40" />
      <div v-else-if="filteredSkills.length" class="grid grid-cols-1 gap-12px sm:grid-cols-2 lg:grid-cols-3">
        <NCard
          v-for="skill in filteredSkills"
          :key="skill.skillId"
          :bordered="false"
          size="small"
          class="card-wrapper shadow-sm transition-shadow hover:shadow-md"
        >
          <div class="flex h-full flex-col gap-8px">
            <div class="flex items-start gap-10px">
              <div class="flex size-40px shrink-0 items-center justify-center rounded-12px bg-amber-50 dark:bg-amber-900/30">
                <SvgIcon icon="lucide:package" class="text-20px text-amber-600" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-4px">
                  <span class="truncate text-14px font-semibold">{{ skill.name }}</span>
                  <SvgIcon
                    v-if="isSkillAuthorized(skill)"
                    icon="lucide:circle-check"
                    class="shrink-0 text-16px text-emerald-500"
                  />
                </div>
                <code class="block truncate text-12px text-slate-400">v{{ skill.version }} · {{ skill.author || '-' }}</code>
              </div>
            </div>
            <div class="flex flex-wrap gap-4px">
              <NTag size="tiny" :bordered="false" type="primary">{{ skill.category }}</NTag>
              <NTag v-for="tag in skill.tags.slice(0, 3)" :key="tag" size="tiny" :bordered="false">
                {{ tag }}
              </NTag>
            </div>
            <p class="line-clamp-2 min-h-32px text-12px text-slate-400">{{ skill.description }}</p>
            <div class="mt-auto flex items-center justify-between gap-8px">
              <NTag v-if="isSkillAuthorized(skill)" type="success" size="small" :bordered="false">
                {{ $t('page.home.square.authorized') }}
              </NTag>
              <NTag v-else-if="isSkillApplied(skill)" type="warning" size="small" :bordered="false">
                {{ $t('page.home.square.applyPending') }}
              </NTag>
              <NTag v-else-if="skill.requiresApproval" type="warning" size="small" :bordered="false">
                {{ $t('page.home.square.requiresApproval') }}
              </NTag>
              <NTag v-else size="small">{{ $t('page.home.square.notAuthorized') }}</NTag>
              <div class="flex items-center gap-6px">
                <NButton
                  v-if="skill.requiresApproval && !isSkillAuthorized(skill) && !isSkillApplied(skill)"
                  size="tiny"
                  type="primary"
                  :disabled="!identity?.opened"
                  @click="handleApplySkill(skill)"
                >
                  {{ $t('page.home.square.apply') }}
                </NButton>
                <NButton
                  size="tiny"
                  type="primary"
                  ghost
                  :loading="downloadingSkillId === skill.skillId"
                  :disabled="!skill.hasPackage"
                  @click="handleDownloadSkill(skill)"
                >
                  {{ $t('page.home.square.skillDownload') }}
                </NButton>
              </div>
            </div>
          </div>
        </NCard>
      </div>
      <NEmpty v-else class="h-240px justify-center" :description="$t('page.home.square.emptySkills')" />
    </template>

    <!-- 接入信息弹窗：路由名/变体/Base URL/API Key -->
    <NModal
      v-model:show="accessVisible"
      :title="$t('page.home.square.accessTitle')"
      preset="card"
      class="w-560px max-w-90%"
      :mask-closable="true"
    >
      <div v-if="accessModel" class="flex flex-col gap-14px">
        <div>
          <div class="mb-4px text-12px font-medium">
            {{ $t('page.home.square.accessModelKey') }}
            <span class="font-normal text-slate-400">({{ $t('page.home.square.accessModelKeyTip') }})</span>
          </div>
          <div class="flex items-center gap-8px">
            <code class="min-w-0 flex-1 truncate rounded-8px bg-slate-100 px-10px py-6px text-13px dark:bg-slate-700/60">
              {{ accessModel.modelKey }}
            </code>
            <NButton
              size="tiny"
              :type="isCopied('modelKey') ? 'success' : 'default'"
              @click="handleCopy('modelKey', accessModel.modelKey)"
            >
              {{ isCopied('modelKey') ? $t('page.home.square.copied') : $t('page.home.square.copy') }}
            </NButton>
          </div>
        </div>
      </div>

      <!-- MCP 接入信息：接入地址/工具清单/客户端配置 JSON -->
      <div v-else-if="accessMcp" class="flex flex-col gap-14px">
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.home.square.accessMcpUrl') }}</div>
          <div class="flex items-center gap-8px">
            <code class="min-w-0 flex-1 truncate rounded-8px bg-slate-100 px-10px py-6px text-13px dark:bg-slate-700/60">
              {{ accessMcp.mcpUrl }}
            </code>
            <NButton
              size="tiny"
              :type="isCopied('mcpUrl') ? 'success' : 'default'"
              @click="handleCopy('mcpUrl', accessMcp.mcpUrl)"
            >
              {{ isCopied('mcpUrl') ? $t('page.home.square.copied') : $t('page.home.square.copy') }}
            </NButton>
          </div>
        </div>
        <div>
          <div class="mb-4px text-12px font-medium">
            {{ $t('page.home.square.toolsCount', { count: accessMcp.tools.length }) }}
          </div>
          <div class="flex flex-wrap gap-4px">
            <NTag v-for="tool in accessMcp.tools.slice(0, 12)" :key="tool.name" size="tiny" :bordered="false">
              {{ tool.name }}
            </NTag>
            <span v-if="accessMcp.tools.length > 12" class="text-12px text-slate-400">…</span>
          </div>
        </div>
        <div v-if="accessMcp.instructions" class="rounded-8px bg-slate-50 px-10px py-8px text-12px text-slate-500 dark:bg-slate-700/40 dark:text-slate-300">
          {{ accessMcp.instructions }}
        </div>
        <div>
          <div class="mb-4px text-12px font-medium">
            {{ $t('page.home.square.accessMcpConfig') }}
            <span class="font-normal text-slate-400">({{ $t('page.home.square.accessMcpConfigTip') }})</span>
          </div>
          <NButton
            size="tiny"
            :type="isCopied('mcpConfig') ? 'success' : 'default'"
            :disabled="!accessMcp.config"
            @click="handleCopyConfig"
          >
            {{ isCopied('mcpConfig') ? $t('page.home.square.copied') : $t('page.home.square.copyConfig') }}
          </NButton>
        </div>
      </div>

      <!-- Base URL 与 API Key(模型/MCP 共用) -->
      <div v-if="accessModel || accessMcp" class="flex flex-col gap-14px">
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.home.square.accessBaseUrl') }}</div>
          <div class="flex items-center gap-8px">
            <code class="min-w-0 flex-1 truncate rounded-8px bg-slate-100 px-10px py-6px text-13px dark:bg-slate-700/60">
              {{ identity?.gatewayUrl || '-' }}
            </code>
            <NButton
              size="tiny"
              :type="isCopied('gatewayUrl') ? 'success' : 'default'"
              @click="handleCopy('gatewayUrl', identity?.gatewayUrl || '')"
            >
              {{ isCopied('gatewayUrl') ? $t('page.home.square.copied') : $t('page.home.square.copy') }}
            </NButton>
          </div>
        </div>
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.home.square.accessApiKey') }}</div>
          <div class="flex items-center gap-8px">
            <code class="min-w-0 flex-1 truncate rounded-8px bg-slate-100 px-10px py-6px text-13px dark:bg-slate-700/60">
              {{ displayKey || '-' }}
            </code>
            <NButton size="tiny" quaternary circle @click="showFullKey = !showFullKey">
              <template #icon>
                <SvgIcon :icon="showFullKey ? 'lucide:eye-off' : 'lucide:eye'" />
              </template>
            </NButton>
            <NButton
              size="tiny"
              :type="isCopied('apiKey') ? 'success' : 'default'"
              @click="handleCopy('apiKey', fullKey)"
            >
              {{ isCopied('apiKey') ? $t('page.home.square.copied') : $t('page.home.square.copy') }}
            </NButton>
          </div>
        </div>
      </div>
    </NModal>

    <!-- 申请订阅弹窗：展示资源 + 申请理由 -->
    <NModal
      v-model:show="applyVisible"
      :title="$t('page.home.square.applyTitle')"
      preset="card"
      class="w-480px max-w-90%"
      :mask-closable="false"
    >
      <div v-if="applyModel || applyMcp || applySkill" class="flex flex-col gap-14px">
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.home.square.applyModel') }}</div>
          <div class="flex items-center gap-8px">
            <span class="text-14px font-medium">{{ applyModel ? applyModel.name : applyMcp ? applyMcp.name : applySkill?.name }}</span>
            <code class="truncate text-12px text-slate-400">
              {{ applyModel ? applyModel.modelKey : applyMcp ? applyMcp.serverName : `v${applySkill?.version}` }}
            </code>
          </div>
        </div>
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.home.square.applyReason') }}</div>
          <NInput
            v-model:value="applyReason"
            type="textarea"
            :rows="3"
            maxlength="500"
            show-count
            :placeholder="$t('page.home.square.applyReasonPlaceholder')"
          />
        </div>
        <div class="flex justify-end gap-8px">
          <NButton quaternary @click="applyVisible = false">{{ $t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="applySubmitting"
            :disabled="!applyReason.trim()"
            @click="handleSubmitApply"
          >
            {{ $t('page.home.square.applySubmit') }}
          </NButton>
        </div>
      </div>
    </NModal>
  </div>
</template>
