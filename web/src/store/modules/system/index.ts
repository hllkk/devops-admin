import { ref, computed } from 'vue';
import { defineStore } from 'pinia';
import { SetupStoreId } from '@/enum';
import { fetchGetPublicSetting } from '@/service/api/system/setting';
import { getLocale, i18n } from '@/locales';

/**
 * 系统公开配置 store(登录页与全局品牌展示用)。
 *
 * init() 拉取免鉴权的 GET /system/setting/public(幂等,仅首次实际请求);
 * 拉到后应用副作用:
 *   - systemName → 覆盖 i18n system.title,已有 $t('system.title') 的登录页/全局 logo 自动联动(无需改模板)
 *   - faviconUrl → 切换 link[rel=icon] 的 href
 * logoUrl 由 SystemLogo 组件按 store.setting.logoUrl 自行读取渲染。
 */
export const useSystemStore = defineStore(SetupStoreId.System, () => {
  const setting = ref<Api.System.PublicSetting | null>(null);
  let loaded = false;

  function applyEffect(s: Api.System.PublicSetting) {
    if (s.systemName) {
      // 覆盖已知 locale,避免切换语言时 system.title 回退到默认
      const locales = Array.from(new Set([getLocale(), 'zh-CN', 'en-US']));
      locales.forEach(l => {
        i18n.global.mergeLocaleMessage(l, { system: { title: s.systemName } });
      });
    }
    if (s.faviconUrl) {
      document.querySelector('link[rel="icon"]')?.setAttribute('href', s.faviconUrl);
    }
  }

  // 拉取公开系统配置(幂等)。失败也置 loaded,用默认配置兜底,避免每次导航重试。
  async function init() {
    if (loaded) return;
    loaded = true;
    const { data, error } = await fetchGetPublicSetting();
    if (!error && data) {
      setting.value = data;
      applyEffect(data);
    }
  }

  /** 是否有任意第三方登录已启用 */
  const hasAnyThirdPartyLogin = computed(() =>
    Boolean(setting.value?.wecomEnabled || setting.value?.wechatEnabled || setting.value?.giteeEnabled || setting.value?.githubEnabled || setting.value?.dingtalkEnabled)
  );

  /** 是否开放注册 */
  const isRegisterEnabled = computed(() => Boolean(setting.value?.registerEnabled));

  /** 是否开放找回密码 */
  const isResetPwdEnabled = computed(() => Boolean(setting.value?.resetPwdEnabled));

  return { setting, init, hasAnyThirdPartyLogin, isRegisterEnabled, isResetPwdEnabled };
});
