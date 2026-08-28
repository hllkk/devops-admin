<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useClipboard } from '@vueuse/core';
import { $t } from '@/locales';
import {
  fetchGetActiveModels,
  fetchGetMyApplications,
  fetchGetMyIdentity,
  fetchSubmitApplication
} from '@/service/api/gateway';
import { MODEL_CATEGORY_OPTIONS, getProviderIcon } from '@/constants/business/gateway';
import SvgIcon from '@/components/custom/svg-icon.vue';

defineOptions({ name: 'GatewaySquare' });

const { copy: copyText, copied } = useClipboard({ legacy: true, copiedDuring: 2000 });

const isLoading = ref(true);
const models = ref<Api.Gateway.ActiveModel[]>([]);
// identity/my：主 Key 明文+已授权模型+网关接入点(未开通 opened=false 时接入按钮禁用)
const identity = ref<Api.Gateway.MyIdentity | null>(null);
const keyword = ref('');

/** 主 Key 已授权 modelKey 集合(未开通为空集) */
const authorizedKeys = computed(() => new Set(identity.value?.opened ? identity.value.models : []));

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

const fullKey = computed(() => (identity.value?.opened ? identity.value.keyValue : ''));
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
  window.$message?.success($t('page.gateway.square.applySuccess'));
  appliedIds.value.add(String(applyModel.value.modelId));
  applyVisible.value = false;
}

function handleCopy(value: string) {
  if (value) copyText(value);
}

onMounted(async () => {
  const [activeRes, identityRes, appRes] = await Promise.all([
    fetchGetActiveModels(),
    fetchGetMyIdentity(),
    fetchGetMyApplications({ status: 'pending', pageNum: 1, pageSize: 100, params: {} })
  ]);
  if (!activeRes.error && activeRes.data) models.value = activeRes.data;
  if (!identityRes.error && identityRes.data) identity.value = identityRes.data;
  if (!appRes.error && appRes.data?.rows) {
    appliedIds.value = new Set(appRes.data.rows.map(r => String(r.resourceId)));
  }
  isLoading.value = false;
});
</script>

<template>
  <div class="h-full flex-col-stretch overflow-hidden">
    <!-- 页头：标题/说明 + 未开通提示 + 搜索 -->
    <NCard :bordered="false" size="small" class="card-wrapper mb-12px flex-shrink-0">
      <div class="flex flex-wrap items-center gap-12px">
        <div class="min-w-0 flex-1">
          <div class="text-16px font-semibold">{{ $t('page.gateway.square.title') }}</div>
          <div class="mt-2px text-12px text-slate-400">{{ $t('page.gateway.square.subtitle') }}</div>
        </div>
        <NInput
          v-model:value="keyword"
          :placeholder="$t('page.gateway.square.searchPlaceholder')"
          clearable
          class="w-240px"
        >
          <template #prefix>
            <SvgIcon icon="lucide:search" class="text-14px text-slate-400" />
          </template>
        </NInput>
      </div>
      <NAlert v-if="identity && !identity.opened" type="warning" :show-icon="true" class="mt-12px">
        {{ $t('page.gateway.square.noIdentity') }}
      </NAlert>
    </NCard>

    <!-- 卡片网格 -->
    <div class="min-h-0 flex-1 overflow-auto pr-1px">
      <div v-if="isLoading" class="h-240px w-full animate-pulse rounded-12px bg-slate-200/60 dark:bg-slate-700/40" />
      <div
        v-else-if="filteredModels.length"
        class="grid grid-cols-1 gap-12px sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      >
        <NCard
          v-for="model in filteredModels"
          :key="model.modelId"
          :bordered="false"
          size="small"
          class="card-wrapper transition-shadow hover:shadow-md"
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
                {{ $t('page.gateway.square.authorized') }}
              </NTag>
              <NTag v-else-if="isApplied(model)" type="warning" size="small" :bordered="false">
                {{ $t('page.gateway.square.applyPending') }}
              </NTag>
              <NTag v-else-if="model.requiresApproval" type="warning" size="small" :bordered="false">
                {{ $t('page.gateway.square.requiresApproval') }}
              </NTag>
              <NTag v-else size="small">{{ $t('page.gateway.square.notAuthorized') }}</NTag>
              <div class="flex items-center gap-6px">
                <NButton
                  v-if="model.requiresApproval && !isAuthorized(model) && !isApplied(model)"
                  size="tiny"
                  type="primary"
                  :disabled="!identity?.opened"
                  @click="handleApply(model)"
                >
                  {{ $t('page.gateway.square.apply') }}
                </NButton>
                <NButton
                  size="tiny"
                  type="primary"
                  ghost
                  :disabled="!identity?.opened"
                  @click="handleViewAccess(model)"
                >
                  {{ $t('page.gateway.square.viewAccess') }}
                </NButton>
              </div>
            </div>
          </div>
        </NCard>
      </div>
      <NEmpty v-else class="h-240px justify-center" :description="$t('page.gateway.square.empty')" />
    </div>

    <!-- 接入信息弹窗：路由名/变体/Base URL/API Key -->
    <NModal
      v-model:show="accessVisible"
      :title="$t('page.gateway.square.accessTitle')"
      preset="card"
      class="w-560px max-w-90%"
      :mask-closable="true"
    >
      <div v-if="accessModel" class="flex flex-col gap-14px">
        <div>
          <div class="mb-4px text-12px font-medium">
            {{ $t('page.gateway.square.accessModelKey') }}
            <span class="font-normal text-slate-400">({{ $t('page.gateway.square.accessModelKeyTip') }})</span>
          </div>
          <div class="flex items-center gap-8px">
            <code class="min-w-0 flex-1 truncate rounded-8px bg-slate-100 px-10px py-6px text-13px dark:bg-slate-700/60">
              {{ accessModel.modelKey }}
            </code>
            <NButton size="tiny" :type="copied ? 'success' : 'default'" @click="handleCopy(accessModel.modelKey)">
              {{ copied ? $t('page.gateway.square.copied') : $t('page.gateway.square.copy') }}
            </NButton>
          </div>
        </div>
        <div v-if="accessModel.hasAnthropicDeployment">
          <div class="mb-4px text-12px font-medium">
            {{ $t('page.gateway.square.accessModelKeyAnthropic') }}
            <span class="font-normal text-slate-400">({{ $t('page.gateway.square.accessAnthropicTip') }})</span>
          </div>
          <div class="flex items-center gap-8px">
            <code class="min-w-0 flex-1 truncate rounded-8px bg-slate-100 px-10px py-6px text-13px dark:bg-slate-700/60">
              {{ accessModel.modelKeyAnthropic }}
            </code>
            <NButton size="tiny" :type="copied ? 'success' : 'default'" @click="handleCopy(accessModel.modelKeyAnthropic)">
              {{ copied ? $t('page.gateway.square.copied') : $t('page.gateway.square.copy') }}
            </NButton>
          </div>
        </div>
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.gateway.square.accessBaseUrl') }}</div>
          <div class="flex items-center gap-8px">
            <code class="min-w-0 flex-1 truncate rounded-8px bg-slate-100 px-10px py-6px text-13px dark:bg-slate-700/60">
              {{ identity?.gatewayUrl || '-' }}
            </code>
            <NButton
              size="tiny"
              :type="copied ? 'success' : 'default'"
              @click="handleCopy(identity?.gatewayUrl || '')"
            >
              {{ copied ? $t('page.gateway.square.copied') : $t('page.gateway.square.copy') }}
            </NButton>
          </div>
        </div>
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.gateway.square.accessApiKey') }}</div>
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
              {{ copied ? $t('page.gateway.square.copied') : $t('page.gateway.square.copy') }}
            </NButton>
          </div>
        </div>
      </div>
    </NModal>
    <!-- 申请订阅弹窗：展示资源 + 申请理由 -->
    <NModal
      v-model:show="applyVisible"
      :title="$t('page.gateway.square.applyTitle')"
      preset="card"
      class="w-480px max-w-90%"
      :mask-closable="false"
    >
      <div v-if="applyModel" class="flex flex-col gap-14px">
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.gateway.square.applyModel') }}</div>
          <div class="flex items-center gap-8px">
            <span class="text-14px font-medium">{{ applyModel.name }}</span>
            <code class="truncate text-12px text-slate-400">{{ applyModel.modelKey }}</code>
          </div>
        </div>
        <div>
          <div class="mb-4px text-12px font-medium">{{ $t('page.gateway.square.applyReason') }}</div>
          <NInput
            v-model:value="applyReason"
            type="textarea"
            :rows="3"
            maxlength="500"
            show-count
            :placeholder="$t('page.gateway.square.applyReasonPlaceholder')"
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
            {{ $t('page.gateway.square.applySubmit') }}
          </NButton>
        </div>
      </div>
    </NModal>
  </div>
</template>
