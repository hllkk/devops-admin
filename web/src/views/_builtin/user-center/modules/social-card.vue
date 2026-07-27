<script lang="ts" setup>
import { computed, ref } from 'vue';
import { useLoading } from '@sa/hooks';
import { useThemeStore } from '@/store/modules/theme';
import { fetchSocialAuthBinding, fetchSocialAuthUnbinding, fetchSocialList } from '@/service/api/system';
import { $t } from '@/locales';

defineOptions({
  name: 'SocialCard'
});

const socialList = ref<Api.System.Social[]>([]);
const { loading, startLoading, endLoading } = useLoading();
const { loading: btnLoading, startLoading: startBtnLoading, endLoading: endBtnLoading } = useLoading();

/** 获取SSO账户列表 */
async function getSsoUserList() {
  startLoading();
  const { data, error } = await fetchSocialList();
  if (!error) {
    socialList.value = data || [];
  }
  endLoading();
}

/** 绑定SSO账户 */
async function bindSsoAccount(type: Api.System.SocialSource) {
  const { data, error } = await fetchSocialAuthBinding(type);
  if (!error) {
    window.location.href = data;
  }
}

/** 解绑SSO账户 */
async function unbindSsoAccount(socialId: CommonType.IdType) {
  startBtnLoading();
  const { error } = await fetchSocialAuthUnbinding(socialId);
  if (!error) {
    window.$message?.success($t('page.userCenter.social.unbindSuccess'));
    await getSsoUserList();
  }
  endBtnLoading();
}

const themeStore = useThemeStore();

interface SocialSourceItem {
  key: Api.System.SocialSource;
  icon: string;
  color: string;
  name: string;
  /** 企微等"扫码自动建号"来源:不支持在用户中心主动绑定到已有账号,仅展示自动关联状态 */
  autoBind?: boolean;
}

const socialSources = computed<SocialSourceItem[]>(() => {
  const githubColor = themeStore.darkMode ? '#ffffff' : '#010409';
  return [
    { key: 'wechat_open', icon: 'ic:outline-wechat', color: '#44b549', name: $t('page.userCenter.social.wechat') },
    { key: 'wecom', icon: 'ic:outline-wechat', color: '#2B7EF9', name: $t('page.userCenter.social.wecom'), autoBind: true },
    { key: 'gitee', icon: 'simple-icons:gitee', color: '#c71d23', name: 'Gitee' },
    { key: 'github', icon: 'mdi:github', color: githubColor, name: 'GitHub' }
  ];
});

getSsoUserList();

function getSocial(key: string) {
  return socialList.value.find(s => s.source.toLowerCase() === key);
}
</script>

<template>
  <NSpin :show="loading" class="mt-16px">
    <div class="grid grid-cols-1 gap-16px 2xl:grid-cols-3 xl:grid-cols-2">
      <div v-for="source in socialSources" :key="source.key" class="relative">
        <NCard class="h-full transition-all duration-300 hover:shadow-md" :bordered="true">
          <template v-if="getSocial(source.key)">
            <div class="flex flex-col items-center gap-16px">
              <NAvatar round size="large" :src="getSocial(source.key)?.avatar" class="size-80px" />
              <div class="text-center">
                <div class="text-16px font-medium">
                  {{ getSocial(source.key)?.nickName }}
                </div>
                <div class="mt-4px text-12px text-gray-500">
                  {{ $t('page.userCenter.social.bindTime') }}<NTime v-if="getSocial(source.key)?.createTime" :time="new Date(getSocial(source.key)!.createTime)" type="datetime" />
                </div>
              </div>
              <NButton
                type="error"
                size="small"
                :loading="btnLoading"
                @click="unbindSsoAccount(getSocial(source.key)?.id || '')"
              >
                {{ $t('page.userCenter.social.unbind') }}
              </NButton>
            </div>
          </template>
          <template v-else>
            <div class="h-full flex flex-col items-center justify-center gap-16px">
              <SvgIcon
                :icon="source.icon"
                class="size-48px"
                :style="{ color: source.color }"
              />
              <div class="text-16px font-medium">{{ source.name }}</div>
              <NButton v-if="!source.autoBind" type="primary" size="small" @click="bindSsoAccount(source.key)">
                {{ $t('page.userCenter.social.bind') }}
              </NButton>
              <div v-else class="px-12px text-center text-12px text-gray-400">{{ $t('page.userCenter.social.autoBindTip') }}</div>
            </div>
          </template>
        </NCard>
      </div>
    </div>
  </NSpin>
</template>

<style scoped>
.border-primary {
  border-color: var(--primary-color);
}
</style>
