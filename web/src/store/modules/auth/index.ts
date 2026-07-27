import { computed, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import { defineStore } from 'pinia';
import { useLoading } from '@sa/hooks';
import { fetchGetUserInfo, fetchLogin, fetchLogout } from '@/service/api';
import { useRouterPush } from '@/hooks/common/router';
import { localStg } from '@/utils/storage';
import { SetupStoreId } from '@/enum';
import { $t } from '@/locales';
import { useRouteStore } from '../route';
import { useTabStore } from '../tab';
import { useNoticeStore } from '../notice';
import { clearAuthStorage, getToken } from './shared';
import { clearProactiveRefreshTimer, scheduleProactiveRefresh } from '@/service/request/shared';

function storeTokenExpiry(expiresAt: number | undefined) {
  if (expiresAt) {
    localStg.set('tokenExpiresAt', expiresAt);
    scheduleProactiveRefresh(expiresAt);
  }
}

export const useAuthStore = defineStore(SetupStoreId.Auth, () => {
  const route = useRoute();
  const authStore = useAuthStore();
  const routeStore = useRouteStore();
  const tabStore = useTabStore();
  const noticeStore = useNoticeStore();
  const { toLogin, redirectFromLogin } = useRouterPush(false);
  const { loading: loginLoading, startLoading, endLoading } = useLoading();

  const token = ref('');

  const userInfo: Api.Auth.UserInfo = reactive({
    user: undefined,
    roles: [],
    permissions: [],
    defaultRouter: ''
  });
  /** is super role in static route */
  const isStaticSuper = computed(() => {
    const { VITE_AUTH_ROUTE_MODE, VITE_STATIC_SUPER_ROLE } = import.meta.env;

    return VITE_AUTH_ROUTE_MODE === 'static' && userInfo.roles.includes(VITE_STATIC_SUPER_ROLE);
  });

  /** Is login */
  const isLogin = computed(() => Boolean(token.value));

  /** Reset auth store */
  async function resetStore() {
    recordUserId();

    clearProactiveRefreshTimer();
    localStg.remove('tokenExpiresAt');

    try {
      await fetchLogout();
    } catch {
      // token 可能已失效，忽略登出接口错误
    }

    clearAuthStorage();
    authStore.$reset();

    if (!route.meta.constant) {
      await toLogin();
    }

    noticeStore.clearNotice();
    tabStore.cacheTabs();
    routeStore.resetStore();
  }

  async function logout() {
    resetStore();
  }

  /** Record the user ID of the previous login session Used to compare with the current user ID on next login */
  function recordUserId() {
    if (!userInfo.user?.userId) {
      return;
    }

    // Store current user ID locally for next login comparison
    localStg.set('lastLoginUserId', userInfo.user?.userId.toString());
  }

  /**
   * Check if current login user is different from previous login user If different, clear all tabs
   *
   * @returns {boolean} Whether to clear all tabs
   */
  function checkTabClear(): boolean {
    if (!userInfo.user?.userId) {
      return false;
    }

    const lastLoginUserId = localStg.get('lastLoginUserId');

    // Clear all tabs if current user is different from previous user
    if (!lastLoginUserId || lastLoginUserId !== userInfo.user?.userId) {
      localStg.remove('globalTabs');
      tabStore.clearTabs();

      localStg.remove('lastLoginUserId');
      return true;
    }

    localStg.remove('lastLoginUserId');
    return false;
  }

  /**
   * Login（httpOnly cookie 模式：仅取 username/password，社交登录保留联合类型以免编译断裂）
   *
   * @param [redirect=true] Whether to redirect after login. Default is `true`
   */
  async function login(loginForm: Api.Auth.PwdLoginForm | Api.Auth.SocialLoginForm, redirect = true) {
    startLoading();

    // 取 username/password/captchaId/captcha（验证码由登录页在提交前完成并写入 loginForm）
    const { username, password, captchaId, captcha } = loginForm as Api.Auth.PwdLoginForm;
    const { data, error } = await fetchLogin({ username, password, captchaId, captcha });

    if (!error && data) {
      localStg.set('isAuthenticated', true);
      token.value = 'authenticated';
      storeTokenExpiry(data.expiresAt);

      const pass = await getUserInfo();
      if (pass) {
        // 登录页是 constant 路由,登录前守卫走未登录分支提前 return,动态路由从未初始化。
        // 这里在跳转前先把动态路由 addRoute,使 redirectFromLogin 按角色默认路由(name)跳转时
        // 目标路由已注册,避免 vue-router name 解析失败而停在登录页(需刷新才进首页)。
        if (!routeStore.isInitAuthRoute) {
          await routeStore.initAuthRoute();
        }

        const isClear = checkTabClear();
        let needRedirect = redirect;
        if (isClear) {
          needRedirect = false;
        }
        // 登录入口:redirect 优先,其次主角色默认路由,最后 toHome 兜底
        await redirectFromLogin(needRedirect, userInfo.defaultRouter);

        window.$notification?.success({
          title: $t('page.login.common.loginSuccess'),
          content: $t('page.login.common.welcomeBack', { userName: userInfo.user?.userName || '' }),
          duration: 4500
        });
      } else {
        resetStore();
      }
    } else {
      resetStore();
    }

    endLoading();

    // 失败时抛错，让调用方（如登录页 doLogin）能捕获并刷新验证码/重置验证态
    if (error || !data) {
      throw error || new Error('登录失败');
    }
  }

  async function getUserInfo() {
    const { data: info, error } = await fetchGetUserInfo();

    if (!error) {
      // update store
      Object.assign(userInfo, info);

      return true;
    }

    return false;
  }

  async function initUserInfo() {
    const maybeToken = getToken();

    if (maybeToken) {
      token.value = maybeToken;
      const pass = await getUserInfo();

      if (!pass) {
        resetStore();
      }
    }
  }

  return {
    token,
    userInfo,
    isStaticSuper,
    isLogin,
    loginLoading,
    resetStore,
    login,
    logout,
    initUserInfo
  };
});
