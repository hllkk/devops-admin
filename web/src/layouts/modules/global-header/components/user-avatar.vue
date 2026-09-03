<script setup lang="ts">
import { computed, onMounted } from 'vue';
import type { VNode } from 'vue';
import { useBoolean } from '@sa/hooks';
import { useAuthStore } from '@/store/modules/auth';
import { useRouterPush } from '@/hooks/common/router';
import { useSvgIcon } from '@/hooks/common/icon';
import { fetchCheckUpdate } from '@/service/api/system';
import defaultAvatar from '@/assets/imgs/soybean.jpg';
import { $t } from '@/locales';
import AboutDialog from '@/components/custom/about-dialog.vue';

defineOptions({
  name: 'UserAvatar'
});

const authStore = useAuthStore();
const { routerPushByKey, toLogin } = useRouterPush();
const { SvgIconVNode } = useSvgIcon();

const { bool: avatarError, setTrue: setError, setFalse: clearError } = useBoolean(false);
const { bool: aboutVisible, setTrue: showAbout } = useBoolean(false);
// 静默检查到有新版本时点亮「关于」入口的圆点标记
const { bool: hasUpdate, setTrue: markUpdate } = useBoolean(false);

function loginOrRegister() {
  toLogin();
}

function handleAvatarLoad() {
  clearError();
}

function handleAvatarError() {
  setError();
}

type DropdownKey = 'home' | 'user-center' | 'about' | 'logout';

type DropdownOption =
  | {
      key: DropdownKey;
      label: string;
      icon?: () => VNode;
    }
  | {
      type: 'divider';
      key: string;
    };

const options = computed(() => {
  const opts: DropdownOption[] = [
    {
      label: $t('route.home'),
      key: 'home',
      icon: SvgIconVNode({ icon: 'ph:house', fontSize: 18 })
    },
    {
      label: $t('common.userCenter'),
      key: 'user-center',
      icon: SvgIconVNode({ icon: 'ph:user-circle', fontSize: 18 })
    },
    {
      type: 'divider',
      key: 'divider'
    },
    {
      // 有新版本时 label 追加圆点提示；点开「关于」可查看 changelog 并升级
      label: $t('common.about') + (hasUpdate.value ? ' ●' : ''),
      key: 'about',
      icon: SvgIconVNode({ icon: 'ph:info', fontSize: 18 })
    },
    {
      label: $t('common.logout'),
      key: 'logout',
      icon: SvgIconVNode({ icon: 'ph:sign-out', fontSize: 18 })
    }
  ];
  return opts;
});

// 登录后静默检查更新一次（检查失败静默：发布服务器未配置/不可达不打扰）
onMounted(() => {
  if (!authStore.isLogin) return;
  fetchCheckUpdate().then(({ data, error }) => {
    if (error || !data?.hasUpdate) return;
    markUpdate();
    window.$notification?.info({
      title: $t('upgrade.foundNewVersion'),
      content: `${data.version ?? ''} ${data.message}`.trim(),
      duration: 6000
    });
  });
});

function logout() {
  window.$dialog?.info({
    title: $t('common.tip'),
    content: $t('common.logoutConfirm'),
    positiveText: $t('common.confirm'),
    negativeText: $t('common.cancel'),
    onPositiveClick: () => {
      authStore.logout();
    }
  });
}

function handleDropdown(key: DropdownKey) {
  if (key === 'logout') {
    logout();
  } else if (key === 'about') {
    showAbout();
  } else {
    routerPushByKey(key);
  }
}
</script>

<template>
  <NButton v-if="!authStore.isLogin" quaternary @click="loginOrRegister">
    {{ $t('page.login.common.loginOrRegister') }}
  </NButton>
  <NDropdown v-else placement="bottom" trigger="click" :options="options" @select="handleDropdown">
    <div class="flex cursor-pointer items-center rounded-md px-2 py-1 transition-colors duration-300 hover:bg-black/6">
      <div class="flex items-center gap-2" :class="{ 'opacity-50': avatarError }">
        <NAvatar
          v-if="authStore.userInfo.user?.avatar"
          :size="24"
          round
          :src="authStore.userInfo.user?.avatar"
          @load="handleAvatarLoad"
          @error="handleAvatarError"
        />
        <NAvatar v-else :size="32" round :src="defaultAvatar" @load="handleAvatarLoad" @error="handleAvatarError" />
        <span class="max-w-120px truncate text-14px font-medium">
          {{ authStore.userInfo.user?.nickName }}
        </span>
      </div>
    </div>
  </NDropdown>
  <AboutDialog v-model:visible="aboutVisible" />
</template>

<style lang="scss" scoped>
.avatar-wrapper {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  border-radius: 6px;
  transition: all 0.3s ease;
  cursor: pointer;

  &:hover {
    background-color: rgba(0, 0, 0, 0.06);
  }
}

.avatar-container {
  display: flex;
  align-items: center;
  gap: 8px;

  &.avatar-error {
    opacity: 0.5;
  }
}

.user-name {
  font-size: 14px;
  max-width: 120px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
