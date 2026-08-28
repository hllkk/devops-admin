<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useClipboard } from '@vueuse/core';
import { $t } from '@/locales';
import { fetchGetActiveModels, fetchGetMyApplications, fetchSubmitApplication } from '@/service/api/gateway';
import { MODEL_CATEGORY_OPTIONS, getProviderIcon } from '@/constants/business/gateway';
import SvgIcon from '@/components/custom/svg-icon.vue';

defineOptions({ name: 'HomeModelSquarePanel' });

/**
 * home「模型广场」Tab 面板：可见模型浏览 + 接入信息 + 申请订阅。
 * identity 由 home 主页统一加载后传入（与 AI 身份卡/我的资源共享同一份数据，避免重复请求）。
 */
const props = defineProps<{
  identity: Api.Gateway.MyIdentity | null;
}>();

const emit = defineEmits<{
  applied: [];
}>();

const { copy: copyText, copied } = useClipboard({ legacy: true, copiedDuring: 2000 });

const isLoading = ref(true);
const models = ref<Api.Gateway.ActiveModel[]>([]);
const keyword = ref('');

/** 主 Key 已授权 modelKey 集合(未开通为空集) */
const authorizedKeys = computed(() => new Set(props.identity?.opened ? props.identity.models : []));

/** 已有待审批申请的资源ID集合(本地维护 + 加载时从我的申请回填,刷新后状态保持) */
const appliedIds = ref<Set<string>>(new Set());

const filteredModels = computed(() => {
  const kw = keyword.value.trim().toLowerCase();
  if (!kw) return models.value;
  return models.value.filter(
    m => m.name.toLowerCase().includes(kw) || m.modelKey.toLowerCase().includes(kw)
  );
});

function isAuthorized(model: Api.Gateway.ActiveModel) {
  return authorizedKeys.value.has(model.modelKey);
}

function isApplied(model: Api.Gateway.ActiveModel) {
  return appliedIds.value.has(String(model.modelId));
}

function categoryLabel(category: string) {
  const hit = MODEL_CATEGORY_OPTIONS.find(o => o.value === category);
  return hit ? $t(hit.label) : category;
}

// ===== 接入信息弹窗 =====
const accessVisible = ref(false);
const accessModel = ref<Api.Gateway.ActiveModel | null>(null);
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
  showFullKey.value = false;
  accessVisible.value = true;
}

// ===== 申请订阅(需审批模型;P2 资源申请审批) =====
const applyVisible = ref(false);
const applyModel = ref<Api.Gateway.ActiveModel | null>(null);
const applyReason = ref('');
const applySubmitting = ref(false);

function handleApply(model: Api.Gateway.ActiveModel) {
  applyModel.value = model;
  applyReason.value = '';
  applyVisible.value = true;
}

async function handleSubmitApply() {
  if (!applyModel.value || !applyReason.value.trim()) return;
  applySubmitting.value = true;
  const { error } = await fetchSubmitApplication({
    resourceType: 'model',
    resourceId: applyModel.value.modelId,
    reason: applyReason.value.trim()
  });
  applySubmitting.value = false;
  if (error) return;
  window.$message?.success($t('page.home.square.applySuccess'));
  appliedIds.value.add(String(applyModel.value.modelId));
  applyVisible.value = false;
  // 通知 home 刷新「我的申请」列表(新 pending 记录)
  emit('applied');
}

function handleCopy(value: string) {
  if (value) copyText(value);
}

onMounted(async () => {
  const [activeRes, appRes] = await Promise.all([
    fetchGetActiveModels(),
    fetchGetMyApplications({ status: 'pending', pageNum: 1, pageSize: 100, params: {} })
  ]);
  if (!activeRes.error && activeRes.data) models.value = activeRes.data;
  if (!appRes.error && appRes.data?.rows) {
    appliedIds.value = new Set(appRes.data.rows.map(r => String(r.resourceId)));
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
        :placeholder="$t('page.home.square.searchPlaceholder')"
        clearable
        class="w-220px rounded-12px"
      >
        <template #prefix>
          <SvgIcon icon="lucide:search" class="text-14px text-slate-400" />
        </template>
      </NInput>
    </div>

    <NAlert v-if="identity && !identity.opened" type="warning" :show-icon="true">
      {{ $t('page.home.square.noIdentity') }}
    </NAlert>

    <!-- 卡片网格 -->
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
            <NButton size="tiny" :type="copied ? 'success' : 'default'" @click="handleCopy(accessModel.modelKey)">
              {{ copied ? $t('page.home.square.copied') : $t('page.home.square.copy') }}
            </NButton>
          </div>
        </div>
        <div v-if="accessModel.hasAnthropicDeployment">
          <div class="mb-4px text-12px font-medium">
            {{ $t('page.home.square.accessModelKeyAnthropic') }}
            <span class="font-normal text-slate-400">({{ $t('page.home.square.accessAnthropicTip') }})</span>
          </div>
          <div class="flex items-center gap-8px">
            <code class="min-w-0 flex-1 truncate rounded-8px bg-slate-100 px-10px py-6px text-13px dark:bg-slate-700/60">
              {{ accessModel.modelKeyAnthropic }}
            </code>
            <NButton size="tiny" :type="copied ? 'success' : 'default'" @click="handleCopy(accessModel.modelKeyAnthropic)">
              {{ copied ? $t('page.home.square.copied') : $t('page.home.square.copy') }}
            </NButton>
          </div>
        </div>
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.home.square.accessBaseUrl') }}</div>
          <div class="flex items-center gap-8px">
            <code class="min-w-0 flex-1 truncate rounded-8px bg-slate-100 px-10px py-6px text-13px dark:bg-slate-700/60">
              {{ identity?.gatewayUrl || '-' }}
            </code>
            <NButton
              size="tiny"
              :type="copied ? 'success' : 'default'"
              @click="handleCopy(identity?.gatewayUrl || '')"
            >
              {{ copied ? $t('page.home.square.copied') : $t('page.home.square.copy') }}
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
            <NButton size="tiny" :type="copied ? 'success' : 'default'" @click="handleCopy(fullKey)">
              {{ copied ? $t('page.home.square.copied') : $t('page.home.square.copy') }}
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
      <div v-if="applyModel" class="flex flex-col gap-14px">
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.home.square.applyModel') }}</div>
          <div class="flex items-center gap-8px">
            <span class="text-14px font-medium">{{ applyModel.name }}</span>
            <code class="truncate text-12px text-slate-400">{{ applyModel.modelKey }}</code>
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
