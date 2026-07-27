/** The storage namespace */
declare namespace StorageType {
  interface Session {
    /** The theme color */
    themeColor: string;
    // /**
    //  * the theme settings
    //  */
    // themeSettings: App.Theme.ThemeSetting;
    sessionObj: {
      url: string;
      data: any;
      time: number;
    };
  }

  interface Local {
    /** The i18n language */
    lang: App.I18n.LangType;
    /** The token */
    token: string;
    /** Fixed sider with mix-menu */
    mixSiderFixed: CommonType.YesOrNo;
    /** The refresh token */
    refreshToken: string;
    /** The theme color */
    themeColor: string;
    /** The dark mode */
    darkMode: boolean;
    /** The theme settings */
    themeSettings: App.Theme.ThemeSetting;
    /**
     * The override theme flags
     *
     * The value is the build time of the project
     */
    overrideThemeFlag: string;
    /** The global tabs */
    globalTabs: App.Global.Tab[];
    /** The backup theme setting before is mobile */
    backupThemeSettingBeforeIsMobile: {
      layout: UnionKey.ThemeLayoutMode;
      siderCollapse: boolean;
    };
    /** The last login user id */
    lastLoginUserId: string;
    /** The login form rember */
    loginRember: string;
    /** httpOnly cookie 模式登录态信号：true 表示已登录（token 本体仅存 cookie） */
    isAuthenticated: boolean;
    /** 上次所在的业务模块（全局公共页跟随来源模块用；非法值由 route store 校验回落） */
    lastModule: string;
    /** access token 过期毫秒时间戳（供主动刷新定时器使用） */
    tokenExpiresAt: number;
    /**
     * Per-module structural theme overrides.
     *
     * Key format: `themeSettings__<module>` (e.g. `themeSettings__disk`).
     * Stores only structural fields (layout / tab / sider / footer / header / page / fixedHeaderAndTab);
     * appearance stays global under `themeSettings`.
     */
    [key: `themeSettings__${string}`]: Partial<App.Theme.ThemeSetting>;
  }
}
