import { computed, effectScope, onScopeDispose, ref, toRefs, watch } from 'vue';
import type { Ref } from 'vue';
import { useDateFormat, useEventListener, useNow, usePreferredColorScheme } from '@vueuse/core';
import { defineStore } from 'pinia';
import { getPaletteColorByNumber } from '@sa/color';
import { localStg } from '@/utils/storage';
import { SetupStoreId } from '@/enum';
import { useAuthStore } from '../auth';
import { useRouteStore } from '../route';
import { DEFAULT_MODULE } from '@/constants/module';
import {
  addThemeVarsToGlobal,
  applyStructural,
  createThemeToken,
  getNaiveTheme,
  initThemeSettings,
  loadModuleOverrides,
  pickStructural,
  toggleAuxiliaryColorModes,
  toggleCssDarkMode,
  type StructuralThemeSetting
} from './shared';

/** Theme store */
export const useThemeStore = defineStore(SetupStoreId.Theme, () => {
  const scope = effectScope();
  const osTheme = usePreferredColorScheme();
  const authStore = useAuthStore();

  /** Theme settings */
  const settings: Ref<App.Theme.ThemeSetting> = ref(initThemeSettings());

  const routeStore = useRouteStore();

  /**
   * 对外生效的菜单模式:disk 模块定死 vertical-mix(disk-layout 复用 base 外壳但固定菜单模式),
   * 其余模块用用户主题设置。base-layout/global-menu/global-sider 统一读它,避免 disk 下外壳与菜单组件模式不一致。
   */
  const effectiveLayoutMode = computed<UnionKey.ThemeLayoutMode>(() =>
    routeStore.currentModule === 'disk' ? 'vertical-mix' : settings.value.layout.mode
  );

  /**
   * Global (admin / default module) structural snapshot — the base every module inherits.
   * Appearance lives directly in `settings` and is always global; only structure is per-module.
   */
  const globalStructural = ref<StructuralThemeSetting>(pickStructural(settings.value));

  /** Per-module structural overrides (non-default modules), persisted to `themeSettings__<module>` */
  const moduleOverrides = ref<Record<string, StructuralThemeSetting>>(loadModuleOverrides());

  // 初始化即对齐当前模块结构:刷新停在非默认模块时,不必依赖 currentModule 首次变化才触发 watch
  applyStructural(
    settings.value,
    routeStore.currentModule === DEFAULT_MODULE
      ? globalStructural.value
      : moduleOverrides.value[routeStore.currentModule] ?? globalStructural.value
  );

  // Module switch: save the outgoing module's current structure, apply the incoming module's structure.
  // Appearance fields are never touched here → appearance stays global across modules (no tearing).
  watch(
    () => routeStore.currentModule,
    (newM, oldM) => {
      if (oldM === DEFAULT_MODULE) {
        globalStructural.value = pickStructural(settings.value);
      } else {
        moduleOverrides.value[oldM] = pickStructural(settings.value);
      }
      const incoming =
        newM === DEFAULT_MODULE ? globalStructural.value : (moduleOverrides.value[newM] ?? globalStructural.value);
      applyStructural(settings.value, incoming);
    }
  );

  // Persist structural edits back to the active scope: global if on default module, else current module override.
  watch(
    () => pickStructural(settings.value),
    structural => {
      const m = routeStore.currentModule;
      if (m === DEFAULT_MODULE) {
        globalStructural.value = structural;
      } else {
        moduleOverrides.value[m] = structural;
      }
    },
    { deep: true }
  );

  /** Optional NaiveUI theme overrides from preset */
  const naiveThemeOverrides: Ref<App.Theme.NaiveUIThemeOverride | undefined> = ref(undefined);

  /** Watermark time instance with controls */
  const { now: watermarkTime, pause: pauseWatermarkTime, resume: resumeWatermarkTime } = useNow({ controls: true });

  /** Dark mode */
  const darkMode = computed(() => {
    if (settings.value.themeScheme === 'auto') {
      return osTheme.value === 'dark';
    }
    return settings.value.themeScheme === 'dark';
  });

  /** grayscale mode */
  const grayscaleMode = computed(() => settings.value.grayscale);

  /** colourWeakness mode */
  const colourWeaknessMode = computed(() => settings.value.colourWeakness);

  /** Theme colors */
  const themeColors = computed(() => {
    const { themeColor, otherColor, isInfoFollowPrimary } = settings.value;
    const colors: App.Theme.ThemeColor = {
      primary: themeColor,
      ...otherColor,
      info: isInfoFollowPrimary ? themeColor : otherColor.info
    };
    return colors;
  });

  /** Naive theme */
  const naiveTheme = computed(() => getNaiveTheme(themeColors.value, settings.value, naiveThemeOverrides.value));

  /**
   * Settings json
   *
   * It is for copy settings
   */
  const settingsJson = computed(() => JSON.stringify(settings.value));

  /** Watermark time date formatter */
  const formattedWatermarkTime = computed(() => {
    const { watermark } = settings.value;
    const date = useDateFormat(watermarkTime, watermark.timeFormat);
    return date.value;
  });

  /** Watermark content */
  const watermarkContent = computed(() => {
    const { watermark } = settings.value;

    if (watermark.enableUserName && authStore.userInfo.user?.userName) {
      return authStore.userInfo.user?.userName;
    }

    if (watermark.enableTime) {
      return formattedWatermarkTime.value;
    }

    return watermark.text;
  });

  /** Reset store */
  function resetStore() {
    const themeStore = useThemeStore();

    themeStore.$reset();
  }

  /**
   * Set theme scheme
   *
   * @param themeScheme
   */
  function setThemeScheme(themeScheme: UnionKey.ThemeScheme) {
    settings.value.themeScheme = themeScheme;
  }

  /**
   * Set grayscale value
   *
   * @param isGrayscale
   */
  function setGrayscale(isGrayscale: boolean) {
    settings.value.grayscale = isGrayscale;
  }

  /**
   * Set colourWeakness value
   *
   * @param isColourWeakness
   */
  function setColourWeakness(isColourWeakness: boolean) {
    settings.value.colourWeakness = isColourWeakness;
  }

  /** Toggle theme scheme */
  function toggleThemeScheme() {
    const themeSchemes: UnionKey.ThemeScheme[] = ['light', 'dark', 'auto'];

    const index = themeSchemes.findIndex(item => item === settings.value.themeScheme);

    const nextIndex = index === themeSchemes.length - 1 ? 0 : index + 1;

    const nextThemeScheme = themeSchemes[nextIndex];

    setThemeScheme(nextThemeScheme);
  }

  /**
   * Update theme colors
   *
   * @param key Theme color key
   * @param color Theme color
   */
  function updateThemeColors(key: App.Theme.ThemeColorKey, color: string) {
    let colorValue = color;

    if (settings.value.recommendColor) {
      // get a color palette by provided color and color name, and use the suitable color

      colorValue = getPaletteColorByNumber(color, 500, true);
    }

    if (key === 'primary') {
      settings.value.themeColor = colorValue;
    } else {
      settings.value.otherColor[key] = colorValue;
    }
  }

  /**
   * Set theme layout
   *
   * @param mode Theme layout mode
   */
  function setThemeLayout(mode: UnionKey.ThemeLayoutMode) {
    settings.value.layout.mode = mode;
  }

  /** Setup theme vars to global */
  function setupThemeVarsToGlobal() {
    const { themeTokens, darkThemeTokens } = createThemeToken(
      themeColors.value,
      settings.value.tokens,
      settings.value.recommendColor
    );
    addThemeVarsToGlobal(themeTokens, darkThemeTokens);
  }

  /**
   * Set watermark enable user name
   *
   * @param enable Whether to enable user name watermark
   */
  function setWatermarkEnableUserName(enable: boolean) {
    settings.value.watermark.enableUserName = enable;

    if (enable) {
      settings.value.watermark.enableTime = false;
    }
  }

  /**
   * Set watermark enable time
   *
   * @param enable Whether to enable time watermark
   */
  function setWatermarkEnableTime(enable: boolean) {
    settings.value.watermark.enableTime = enable;

    if (enable) {
      settings.value.watermark.enableUserName = false;
    }
  }

  /**
   * Set NaiveUI theme overrides
   *
   * @param overrides NaiveUI theme overrides or undefined to clear
   */
  function setNaiveThemeOverrides(overrides?: App.Theme.NaiveUIThemeOverride) {
    naiveThemeOverrides.value = overrides;
  }

  /** Only run timer when watermark is visible and time display is enabled */
  function updateWatermarkTimer() {
    const { watermark } = settings.value;
    const shouldRunTimer = watermark.visible && watermark.enableTime;

    if (shouldRunTimer) {
      resumeWatermarkTime();
    } else {
      pauseWatermarkTime();
    }
  }

  /** Cache theme settings (global = appearance + admin structural; plus per-module structural overrides) */
  function cacheThemeSettings() {
    const isProd = import.meta.env.PROD;

    if (!isProd) return;

    // Global: appearance (always global, taken from current settings) + admin/default structural.
    // Using globalStructural (not current settings' structure) avoids polluting global when the user
    // is on a non-default module at unload time.
    localStg.set('themeSettings', { ...settings.value, ...globalStructural.value });

    // Per-module structural overrides
    Object.entries(moduleOverrides.value).forEach(([m, structural]) => {
      localStg.set(`themeSettings__${m}`, structural);
    });
  }

  // cache theme settings when page is closed or refreshed
  useEventListener(window, 'beforeunload', () => {
    cacheThemeSettings();
  });

  // watch store
  scope.run(() => {
    // watch dark mode
    watch(
      darkMode,
      val => {
        toggleCssDarkMode(val);
        localStg.set('darkMode', val);
      },
      { immediate: true }
    );

    watch(
      [grayscaleMode, colourWeaknessMode],
      val => {
        toggleAuxiliaryColorModes(val[0], val[1]);
      },
      { immediate: true }
    );

    // themeColors change, update css vars and storage theme color
    watch(
      themeColors,
      val => {
        setupThemeVarsToGlobal();
        localStg.set('themeColor', val.primary);
      },
      { immediate: true }
    );

    // watch watermark settings to control timer
    watch(
      () => [settings.value.watermark.visible, settings.value.watermark.enableTime],
      () => {
        updateWatermarkTimer();
      },
      { immediate: true }
    );
  });

  /** On scope dispose */
  onScopeDispose(() => {
    scope.stop();
  });

  return {
    ...toRefs(settings.value),
    effectiveLayoutMode,
    darkMode,
    themeColors,
    naiveTheme,
    settingsJson,
    watermarkContent,
    setGrayscale,
    setColourWeakness,
    resetStore,
    setThemeScheme,
    toggleThemeScheme,
    updateThemeColors,
    setThemeLayout,
    setWatermarkEnableUserName,
    setWatermarkEnableTime,
    setNaiveThemeOverrides
  };
});
