/** The global namespace for the app */
declare namespace App {
  /** Theme namespace */
  namespace Theme {
    type ColorPaletteNumber = import('@sa/color').ColorPaletteNumber;

    /** NaiveUI theme overrides that can be specified in preset */
    type NaiveUIThemeOverride = import('naive-ui').GlobalThemeOverrides;

    /** Theme setting */
    interface ThemeSetting {
      /** Theme scheme */
      themeScheme: UnionKey.ThemeScheme;
      /** grayscale mode */
      grayscale: boolean;
      /** colour weakness mode */
      colourWeakness: boolean;
      /** Whether to recommend color */
      recommendColor: boolean;
      /** Theme color */
      themeColor: string;
      /** Theme radius */
      themeRadius: number;
      /** Other color */
      otherColor: OtherColor;
      /** Whether info color is followed by the primary color */
      isInfoFollowPrimary: boolean;
      /** Layout */
      layout: {
        /** Layout mode */
        mode: UnionKey.ThemeLayoutMode;
        /** Scroll mode */
        scrollMode: UnionKey.ThemeScrollMode;
      };
      /** Page */
      page: {
        /** Whether to show the page transition */
        animate: boolean;
        /** Page animate mode */
        animateMode: UnionKey.ThemePageAnimateMode;
      };
      /** Header */
      header: {
        /** Header height */
        height: number;
        /** Header breadcrumb */
        breadcrumb: {
          /** Whether to show the breadcrumb */
          visible: boolean;
          /** Whether to show the breadcrumb icon */
          showIcon: boolean;
        };
        /** Multilingual */
        multilingual: {
          /** Whether to show the multilingual */
          visible: boolean;
        };
        globalSearch: {
          /** Whether to show the GlobalSearch */
          visible: boolean;
        };
      };
      /** Tab */
      tab: {
        /** Whether to show the tab */
        visible: boolean;
        /**
         * Whether to cache the tab
         *
         * If cache, the tabs will get from the local storage when the page is refreshed
         */
        cache: boolean;
        /** Tab height */
        height: number;
        /** Tab mode */
        mode: UnionKey.ThemeTabMode;
        /** Whether to close tab by middle click */
        closeTabByMiddleClick: boolean;
      };
      /** Fixed header and tab */
      fixedHeaderAndTab: boolean;
      /** Sider */
      sider: {
        /** Inverted sider */
        inverted: boolean;
        /** Sider width */
        width: number;
        /** Collapsed sider width */
        collapsedWidth: number;
        /** Sider width when the layout is 'vertical-mix', 'top-hybrid-sidebar-first', or 'top-hybrid-header-first' */
        mixWidth: number;
        /**
         * Collapsed sider width when the layout is 'vertical-mix', 'top-hybrid-sidebar-first', or
         * 'top-hybrid-header-first'
         */
        mixCollapsedWidth: number;
        /** Child menu width when the layout is 'vertical-mix', 'top-hybrid-sidebar-first', or 'top-hybrid-header-first' */
        mixChildMenuWidth: number;
        /** Whether to auto select the first submenu */
        autoSelectFirstMenu: boolean;
      };
      /** Footer */
      footer: {
        /** Whether to show the footer */
        visible: boolean;
        /** Whether fixed the footer */
        fixed: boolean;
        /** Footer height */
        height: number;
        /**
         * Whether float the footer to the right when the layout is 'top-hybrid-sidebar-first' or
         * 'top-hybrid-header-first'
         */
        right: boolean;
      };
      /** Watermark */
      watermark: {
        /** Whether to show the watermark */
        visible: boolean;
        /** Watermark text */
        text: string;
        /** Whether to use user name as watermark text */
        enableUserName: boolean;
        /** Whether to use current time as watermark text */
        enableTime: boolean;
        /** Time format for watermark text */
        timeFormat: string;
      };
      /** define some theme settings tokens, will transform to css variables */
      tokens: {
        light: ThemeSettingToken;
        dark?: {
          [K in keyof ThemeSettingToken]?: Partial<ThemeSettingToken[K]>;
        };
      };
    }

    interface OtherColor {
      info: string;
      success: string;
      warning: string;
      error: string;
    }

    interface ThemeColor extends OtherColor {
      primary: string;
    }

    type ThemeColorKey = keyof ThemeColor;

    type ThemePaletteColor = {
      [key in ThemeColorKey | `${ThemeColorKey}-${ColorPaletteNumber}`]: string;
    };

    type BaseToken = Record<string, Record<string, string>>;

    interface ThemeSettingTokenColor {
      /** the progress bar color, if not set, will use the primary color */
      nprogress?: string;
      container: string;
      layout: string;
      inverted: string;
      'base-text': string;
    }

    interface ThemeSettingTokenBoxShadow {
      header: string;
      sider: string;
      tab: string;
    }

    interface ThemeSettingToken {
      colors: ThemeSettingTokenColor;
      boxShadow: ThemeSettingTokenBoxShadow;
    }

    type ThemeTokenColor = ThemePaletteColor & ThemeSettingTokenColor;

    /** Theme token CSS variables */
    type ThemeTokenCSSVars = {
      colors: ThemeTokenColor & { [key: string]: string };
      boxShadow: ThemeSettingTokenBoxShadow & { [key: string]: string };
    };
  }

  /** Global namespace */
  namespace Global {
    type VNode = import('vue').VNode;
    type RouteLocationNormalizedLoaded = import('vue-router').RouteLocationNormalizedLoaded;
    type RouteKey = import('@elegant-router/types').RouteKey;
    type RouteMap = import('@elegant-router/types').RouteMap;
    type RoutePath = import('@elegant-router/types').RoutePath;
    type LastLevelRouteKey = import('@elegant-router/types').LastLevelRouteKey;

    /** The router push options */
    type RouterPushOptions = {
      query?: Record<string, string>;
      params?: Record<string, string>;
      force?: boolean;
    };

    /** The global header props */
    interface HeaderProps {
      /** Whether to show the logo */
      showLogo?: boolean;
      /** Whether to show the menu toggler */
      showMenuToggler?: boolean;
      /** Whether to show the menu */
      showMenu?: boolean;
    }

    /** The global menu */
    type Menu = {
      /**
       * The menu key
       *
       * Equal to the route key
       */
      key: string;
      /** The menu label */
      label: string;
      /** The menu i18n key */
      i18nKey?: I18n.I18nKey | null;
      /** The route key */
      routeKey: RouteKey;
      /** The route path */
      routePath: RoutePath;
      /** The menu icon */
      icon?: () => VNode;
      /** The menu children */
      children?: Menu[];
    };

    type Breadcrumb = Omit<Menu, 'children'> & {
      options?: Breadcrumb[];
    };

    /** Tab route */
    type TabRoute = Pick<RouteLocationNormalizedLoaded, 'name' | 'path' | 'meta'> &
      Partial<Pick<RouteLocationNormalizedLoaded, 'fullPath' | 'query' | 'matched'>>;

    /** The global tab */
    type Tab = {
      /** The tab id */
      id: string;
      /** The tab label */
      label: string;
      /**
       * The new tab label
       *
       * If set, the tab label will be replaced by this value
       */
      newLabel?: string;
      /**
       * The old tab label
       *
       * when reset the tab label, the tab label will be replaced by this value
       */
      oldLabel?: string;
      /** The tab route key */
      routeKey: LastLevelRouteKey;
      /** The tab route path */
      routePath: RouteMap[LastLevelRouteKey];
      /** The tab route full path */
      fullPath: string;
      /** The tab fixed index */
      fixedIndex?: number | null;
      /**
       * Tab icon
       *
       * Iconify icon
       */
      icon?: string;
      /**
       * Tab local icon
       *
       * Local icon
       */
      localIcon?: string;
      /** I18n key */
      i18nKey?: I18n.I18nKey | null;
    };

    /** Form rule */
    type FormRule = import('naive-ui').FormItemRule;

    /** The global dropdown key */
    type DropdownKey = 'closeCurrent' | 'closeOther' | 'closeLeft' | 'closeRight' | 'closeAll' | 'pin' | 'unpin';
  }

  /**
   * I18n namespace
   *
   * Locales type
   */
  namespace I18n {
    type RouteKey = import('@elegant-router/types').RouteKey;

    type LangType = 'en-US' | 'zh-CN';

    type LangOption = {
      label: string;
      key: LangType;
    };

    type I18nRouteKey = Exclude<RouteKey, 'root' | 'not-found'>;

    type FormMsg = {
      required: string;
      invalid: string;
    };

    type Schema = {
      system: {
        title: string;
        updateTitle: string;
        updateContent: string;
        updateConfirm: string;
        updateCancel: string;
      };
      common: {
        action: string;
        add: string;
        addSuccess: string;
        backToHome: string;
        batchDelete: string;
        cancel: string;
        close: string;
        check: string;
        selectAll: string;
        expandColumn: string;
        columnSetting: string;
        config: string;
        confirm: string;
        delete: string;
        deleteSuccess: string;
        confirmDelete: string;
        edit: string;
        warning: string;
        error: string;
        index: string;
        keywordSearch: string;
        logout: string;
        logoutConfirm: string;
        lookForward: string;
        modify: string;
        modifySuccess: string;
        noData: string;
        operate: string;
        pleaseCheckValue: string;
        refresh: string;
        reset: string;
        search: string;
        switch: string;
        tip: string;
        trigger: string;
        save: string;
        update: string;
        updateSuccess: string;
        updateExisting: string;
        export: string;
        import: string;
        importSuccess: string;
        importFail: string;
        importTemplate: string;
        importTip: string;
        importSize: string;
        importFormat: string;
        importEnd: string;
        importResult: string;
        downloadTemplate: string;
        userCenter: string;
        selected: string;
        anyRecords: string;
        clear: string;
        noSelectRecord: string;
        copySuccess: string;
        copyNotSupported: string;
        login: string;
        noChange: string;
        yesOrNo: {
          yes: string;
          no: string;
        };
      };
      request: {
        logout: string;
        logoutMsg: string;
        logoutWithModal: string;
        logoutWithModalMsg: string;
        refreshToken: string;
        tokenExpired: string;
      };
      theme: {
        themeDrawerTitle: string;
        tabs: {
          appearance: string;
          layout: string;
          general: string;
          preset: string;
        };
        appearance: {
          themeSchema: { title: string } & Record<UnionKey.ThemeScheme, string>;
          grayscale: string;
          colourWeakness: string;
          themeColor: {
            title: string;
            followPrimary: string;
          } & Record<Theme.ThemeColorKey, string>;
          recommendColor: string;
          recommendColorDesc: string;
          themeRadius: {
            title: string;
          };
          preset: {
            title: string;
            apply: string;
            applySuccess: string;
            [key: string]:
              | {
                  name: string;
                  desc: string;
                }
              | string;
          };
        };
        layout: {
          layoutMode: { title: string } & Record<UnionKey.ThemeLayoutMode, string> & {
              [K in `${UnionKey.ThemeLayoutMode}_detail`]: string;
            };
          tab: {
            title: string;
            visible: string;
            cache: string;
            cacheTip: string;
            height: string;
            mode: { title: string } & Record<UnionKey.ThemeTabMode, string>;
            closeByMiddleClick: string;
            closeByMiddleClickTip: string;
          };
          header: {
            title: string;
            height: string;
            breadcrumb: {
              visible: string;
              showIcon: string;
            };
          };
          sider: {
            title: string;
            inverted: string;
            width: string;
            collapsedWidth: string;
            mixWidth: string;
            mixCollapsedWidth: string;
            mixChildMenuWidth: string;
            autoSelectFirstMenu: string;
            autoSelectFirstMenuTip: string;
          };
          footer: {
            title: string;
            visible: string;
            fixed: string;
            height: string;
            right: string;
          };
          content: {
            title: string;
            scrollMode: { title: string; tip: string } & Record<UnionKey.ThemeScrollMode, string>;
            page: {
              animate: string;
              mode: { title: string } & Record<UnionKey.ThemePageAnimateMode, string>;
            };
            fixedHeaderAndTab: string;
          };
        };
        general: {
          title: string;
          watermark: {
            title: string;
            visible: string;
            text: string;
            enableUserName: string;
            enableTime: string;
            timeFormat: string;
          };
          multilingual: {
            title: string;
            visible: string;
          };
          globalSearch: {
            title: string;
            visible: string;
          };
        };
        configOperation: {
          copyConfig: string;
          copySuccessMsg: string;
          resetConfig: string;
          resetSuccessMsg: string;
        };
      };
      route: Record<I18nRouteKey, string>;
      /** 业务模块显示名（非路由，独立于 route 命名空间以免受 I18nRouteKey 约束） */
      module: {
        admin: string;
        disk: string;
        server: string;
        gateway: string;
      };
      /** 字典数据值的动态国际化键：dict.{dictType}.{value}，与后端 dictLabel 约定一致 */
      dict: {
        [dictType: string]: {
          [dictValue: string]: string;
        };
      };
      page: {
        login: {
          common: {
            title: string;
            subTitle: string;
            loginOrRegister: string;
            userNamePlaceholder: string;
            phonePlaceholder: string;
            codePlaceholder: string;
            passwordPlaceholder: string;
            confirmPasswordPlaceholder: string;
            codeLogin: string;
            confirm: string;
            back: string;
            validateSuccess: string;
            loginSuccess: string;
            welcomeBack: string;
            register: string;
          };
          pwdLogin: {
            title: string;
            rememberMe: string;
            forgetPassword: string;
            register: string;
            otherAccountLogin: string;
            otherLoginMode: string;
            superAdmin: string;
            admin: string;
            user: string;
          };
          codeLogin: {
            title: string;
            getCode: string;
            reGetCode: string;
            sendCodeSuccess: string;
            imageCodePlaceholder: string;
          };
          register: {
            title: string;
            agreement: string;
            protocol: string;
            policy: string;
          };
          resetPwd: {
            title: string;
          };
          bindWeChat: {
            title: string;
          };
          captcha: {
            title: string;
            clickTitle: string;
            slideTitle: string;
            rotateTitle: string;
            loginWithCaptcha: string;
            imagePlaceholder: string;
            imageRequired: string;
            refresh: string;
          };
        };
        init: {
          title: string;
          subTitle: string;
          noticeTitle: string;
          noticeDesc: string;
          confirm: string;
          back: string;
          form: {
            adminPassword: string;
            adminPasswordPlaceholder: string;
            dbType: string;
            host: string;
            hostPlaceholder: string;
            port: string;
            portPlaceholder: string;
            userName: string;
            userNamePlaceholder: string;
            password: string;
            passwordPlaceholder: string;
            dbName: string;
            dbNamePlaceholder: string;
            dbPath: string;
            dbPathPlaceholder: string;
            template: string;
            templatePlaceholder: string;
            redisAddr: string;
            redisAddrPlaceholder: string;
            redisPassword: string;
            redisPasswordPlaceholder: string;
            redisDB: string;
            redisDBPlaceholder: string;
          };
          step: {
            db: string;
            redis: string;
            admin: string;
          };
          rule: {
            adminPasswordLength: string;
            redisAddrRequired: string;
          };
          testConnection: string;
          testing: string;
          testConnectionSuccess: string;
          testConnectionFailed: string;
          next: string;
          prev: string;
          finish: string;
          submit: string;
          submitting: string;
          successTitle: string;
          toLogin: string;
        };
        home: {
          greeting: string;
          weatherDesc: string;
          projectCount: string;
          todo: string;
          message: string;
          downloadCount: string;
          registerCount: string;
          schedule: string;
          study: string;
          work: string;
          rest: string;
          entertainment: string;
          visitCount: string;
          turnover: string;
          dealCount: string;
          projectNews: {
            title: string;
            moreNews: string;
            desc1: string;
            desc2: string;
            desc3: string;
            desc4: string;
            desc5: string;
          };
          creativity: string;
        };
        system: {
          user: {
            title: string;
            userName: string;
            nickName: string;
            sex: string;
            roleIds: string;
            postIds: string;
            deptName: string;
            email: string;
            phonenumber: string;
            status: string;
            avatar: string;
            remark: string;
            createTime: string;
            password: string;
            confirmPassword: string;
            statusChangeSuccess: string;
            addUser: string;
            editUser: string;
            userList: string;
            form: {
              userName: FormMsg;
              nickName: FormMsg;
              deptId: FormMsg;
              phonenumber: FormMsg;
              status: FormMsg;
              password: FormMsg;
              confirmPassword: FormMsg;
              sex: FormMsg;
              email: FormMsg;
              roleIds: FormMsg;
              postIds: FormMsg;
              remark: FormMsg;
            };
          };
          setting: {
            general: string;
            security: string;
            generalDesc: string;
            securityDesc: string;
            ldap: string;
            ldapDesc: string;
            disk: string;
            diskDesc: string;
            notify: string;
            notifyDesc: string;
            auth: string;
            authDesc: string;
            save: string;
            saveSuccess: string;
            saveFail: string;
            loadFail: string;
            systemName: string;
            systemDescription: string;
            logoUrl: string;
            faviconUrl: string;
            defaultPassword: string;
            defaultPasswordPlaceholder: string;
            userDefaultPassword: string;
            userDefaultRole: string;
            captchaTitle: string;
            enableVerifyCode: string;
            verifyCodeType: string;
            verifyCodeLen: string;
            verifyCodeExp: string;
            verifyCodeTokenExp: string;
            verifyInaccuracy: string;
            logTitle: string;
            loginLogRetentionDays: string;
            operationLogRetentionDays: string;
            captchaType: { image: string; click: string; slide: string; dragdrop: string; rotate: string };
            passwordTitle: string;
            passwordMinLength: string;
            passwordRequireUppercase: string;
            passwordRequireLowercase: string;
            passwordRequireDigit: string;
            passwordRequireSpecial: string;
            loginFailLockCount: string;
            loginFailLockTime: string;
            ipTitle: string;
            ipValidationEnabled: string;
            ipValidationMode: string;
            ipBlacklist: string;
            ipWhitelist: string;
            ipListPlaceholder: string;
            ipMode: { blacklist: string; whitelist: string };
            unitMinute: string;
            unitDay: string;
            unitChar: string;
            unitPixel: string;
            unitSecond: string;
            unitTimes: string;
            tabCaptcha: string;
            tabPassword: string;
            tabLimit: string;
            tabLock: string;
            tabExpire: string;
            tabAccess: string;
            tabAccount: string;
            tabGeneral: string;
            tabLog: string;
            tabLdap: string;
            tabDisk: string;
            tabNotify: string;
            tabAuth: string;
            captchaEnabled: string;
            captchaTypeLabel: string;
            captchaOpen: string;
            captchaTimeout: string;
            captchaTolerance: string;
            keyLong: string;
            imgWidth: string;
            imgHeight: string;
            limitEnable: string;
            limitWindow: string;
            limitCount: string;
            pwdExpireEnable: string;
            pwdExpireDays: string;
            // LDAP 配置
            ldapEnabled: string;
            ldapHost: string;
            ldapHostPlaceholder: string;
            ldapPort: string;
            ldapUseSSL: string;
            ldapBindDN: string;
            ldapBindDNPlaceholder: string;
            ldapBindPass: string;
            ldapBindPassPlaceholder: string;
            ldapBaseDN: string;
            ldapBaseDNPlaceholder: string;
            ldapFilter: string;
            ldapFilterPlaceholder: string;
            ldapAttrUsername: string;
            ldapAttrNickname: string;
            ldapAttrEmail: string;
            ldapAutoCreate: string;
            ldapAutoCreateTip: string;
            ldapTestConnection: string;
            ldapTestSuccess: string;
            ldapTestFail: string;
            tabLdapConnection: string;
            tabLdapAttrMap: string;
            tabLdapUserPolicy: string;
            // 网盘配置
            diskMaxUploadSize: string;
            diskStorageQuota: string;
            diskAllowedExtensions: string;
            diskAllowedExtensionsTip: string;
            diskBlockedExtensions: string;
            diskBlockedExtensionsTip: string;
            diskRecycleBinRetentionDays: string;
            diskUnitMB: string;
            diskUnitGB: string;
            diskUnitTB: string;
            diskName: string;
            diskNamePlaceholder: string;
            diskLogo: string;
            diskLogoPlaceholder: string;
            diskSectionOnlyOffice: string;
            diskOnlyOfficeEnabled: string;
            diskOnlyOfficeServerUrl: string;
            diskOnlyOfficeServerUrlPlaceholder: string;
            diskOnlyOfficeServerUrlTip: string;
            diskOnlyOfficeTokenSecret: string;
            diskOnlyOfficeTokenSecretPlaceholder: string;
            diskOnlyOfficeTokenSecretTip: string;
            diskOnlyOfficeCallbackUrl: string;
            diskOnlyOfficeCallbackUrlPlaceholder: string;
            diskOnlyOfficeCallbackUrlTip: string;
            tabDiskBasic: string;
            tabDiskDisplay: string;
            tabDiskOnlyOffice: string;
            // 通知配置
            notifyEmailEnabled: string;
            notifyEmailHost: string;
            notifyEmailHostPlaceholder: string;
            notifyEmailPort: string;
            notifyEmailUsername: string;
            notifyEmailUsernamePlaceholder: string;
            notifyEmailPassword: string;
            notifyEmailPasswordPlaceholder: string;
            notifyEmailFromAddr: string;
            notifyEmailFromAddrPlaceholder: string;
            notifyEmailFromName: string;
            notifyEmailFromNamePlaceholder: string;
            notifyEmailSSLMode: string;
            notifyEmailSSLModeNone: string;
            notifyEmailSSLModeSSL: string;
            notifyEmailSSLModeStartTLS: string;
            notifyTestEmail: string;
            notifyTestEmailTo: string;
            notifyTestEmailToPlaceholder: string;
            notifyTestEmailSuccess: string;
            notifyTestEmailFail: string;
            notifyTestEmailSending: string;
            notifyWebhookEnabled: string;
            notifyWebhookUrl: string;
            notifyWebhookUrlPlaceholder: string;
            notifyWebhookSecret: string;
            notifyWebhookSecretPlaceholder: string;
            tabNotifyEmail: string;
            tabNotifyWebhook: string;
            // 认证配置
            tabAccountFunction: string;
            tabAuthWecom: string;
            tabAuthWechat: string;
            tabAuthGitee: string;
            tabAuthGithub: string;
            tabAuthDingtalk: string;
            authRegisterEnabled: string;
            authRegisterEnabledTip: string;
            authResetPwdEnabled: string;
            authResetPwdEnabledTip: string;
            authOAuthEnabled: string;
            authOAuthClientId: string;
            authOAuthClientIdPlaceholder: string;
            authOAuthClientSecret: string;
            authOAuthClientSecretPlaceholder: string;
            authOAuthCallbackUrl: string;
            authOAuthCallbackUrlPlaceholder: string;
            authOAuthCallbackUrlTip: string;
            authWecomCorpId: string;
            authWecomCorpIdPlaceholder: string;
            authWecomAgentId: string;
            authWecomAgentIdPlaceholder: string;
            authWecomDomainVerifyTitle: string;
            authWecomDomainVerifyTip1: string;
            authWecomDomainVerifyTip2: string;
            authWecomDomainVerifyTip3: string;
            authWecomDomainFileName: string;
            authWecomDomainFileNamePlaceholder: string;
            authWecomDomainFileContent: string;
            authWecomDomainFileContentPlaceholder: string;
          };
          role: {
            title: string;
            listTitle: string;
            roleName: string;
            roleKey: string;
            roleKeyShort: string;
            roleSort: string;
            status: string;
            remark: string;
            createTime: string;
            menuPermission: string;
            addRole: string;
            editRole: string;
            assignUser: string;
            assignUserTitle: string;
            statusChangeSuccess: string;
            roleKeyTip: string;
            form: {
              roleId: FormMsg;
              roleName: FormMsg;
              roleKey: FormMsg;
              roleSort: FormMsg;
              status: FormMsg;
              remark: FormMsg;
            };
          };
          menu: {
            title: string;
            parentId: string;
            orderNum: string;
            rootName: string;
            menuName: string;
            addMenu: string;
            addChildMenu: string;
            editMenu: string;
            perms: string;
            permsTip: string;
            status: string;
            statusTip: string;
            emptyMenu: string;
            menuDetail: string;
            menuType: string;
            component: string;
            componentTip: string;
            layout: string;
            layoutTip: string;
            icon: string;
            iconType: string;
            iconifyTip: string;
            path: string;
            pathTip: string;
            externalPath: string;
            query: string;
            iframeQuery: string;
            isFrame: string;
            isFrameTip: string;
            visible: string;
            visibleTip: string;
            isCache: string;
            cache: string;
            isCacheTip: string;
            noCache: string;
            cascadeDelete: string;
            cascadeDeleteContent: string;
            createTime: string;
            buttonPermissionList: string;
            placeholder: {
              localIconPlaceholder: string;
              iconifyIconPlaceholder: string;
              queryKey: string;
              queryValue: string;
              queryIframe: string;
            };
            form: {
              parentId: FormMsg;
              menuIds: FormMsg;
              menuName: FormMsg;
              orderNum: FormMsg;
              perms: FormMsg;
              path: FormMsg;
              component: FormMsg;
            };
          };
          notice: {
            title: string;
            listTitle: string;
            noticeTitle: string;
            noticeType: string;
            noticeContent: string;
            status: string;
            createByName: string;
            createTime: string;
            addNotice: string;
            editNotice: string;
            placeholder: {
              noticeTitle: string;
              noticeType: string;
            };
            form: {
              noticeId: FormMsg;
              noticeTitle: FormMsg;
              noticeType: FormMsg;
              noticeContent: FormMsg;
              status: FormMsg;
            };
          };
          dept: {
            title: string;
            parentId: string;
            deptName: string;
            deptCategory: string;
            leader: string;
            phone: string;
            email: string;
            sort: string;
            status: string;
            createTime: string;
            orderNum: string;
            expandAll: string;
            collapseAll: string;
            empty: string;
            addDept: string;
            editDept: string;
            placeholder: {
              defaultLeaderPlaceHolder: string;
              addDataLeaderPlaceHolder: string;
              deptUserIsEmptyLeaderPlaceHolder: string;
            };
            error: {
              getDeptDataFail: string;
              getDeptUserDataFail: string;
            };
            form: {
              deptId: FormMsg;
              parentId: FormMsg;
              orderNum: FormMsg;
              deptName: FormMsg;
              deptCategory: FormMsg;
              status: FormMsg;
              phone: FormMsg;
              email: FormMsg;
            };
          };
          post: {
            title: string;
            listTitle: string;
            deptTreeTitle: string;
            emptyDept: string;
            exportFileName: string;
            belongDept: string;
            postCode: string;
            postCategory: string;
            postName: string;
            postSort: string;
            status: string;
            remark: string;
            createTime: string;
            addPost: string;
            editPost: string;
            form: {
              postId: FormMsg;
              deptId: FormMsg;
              postCode: FormMsg;
              postCategory: FormMsg;
              postName: FormMsg;
              postSort: FormMsg;
              status: FormMsg;
              remark: FormMsg;
            };
          };
          dict: {
            title: string;
            dictName: string;
            dictTypeTitle: string;
            dictType: string;
            dictData: string;
            status: string;
            addDict: string;
            editDict: string;
            addDictType: string;
            addDictData: string;
            exportDictType: string;
            editDictData: string;
            editDictType: string;
            refreshDictType: string;
            dictTypeIsEmpty: string;
            refreshCache: string;
            confirmDeleteDictType: string;
            refreshCacheSuccess: string;
            remark: string;
            createTime: string;
            data: {
              title: string;
              label: string;
              value: string;
              dictSort: string;
              listClass: string;
              cssClass: string;
              status: string;
              remark: string;
              createTime: string;
              isDefault: string;
            };
            form: {
              dictId: FormMsg;
              dictName: FormMsg;
              dictCode: FormMsg;
              dictLabel: FormMsg;
              dictValue: FormMsg;
              dictType: FormMsg;
              listClass: FormMsg;
              cssClass: FormMsg;
              dictSort: FormMsg;
              status: FormMsg;
              isDefault: FormMsg;
              remark: FormMsg;
            };
          };
          timer: {
            title: string;
            listTitle: string;
            id: string;
            name: string;
            description: string;
            spec: string;
            executorType: string;
            executorMethod: string;
            executorHttp: string;
            withSeconds: string;
            withSecondsHint: string;
            methodName: string;
            methodNameHint: string;
            httpUrl: string;
            httpMethod: string;
            httpHeader: string;
            httpBody: string;
            httpAllowPrivate: string;
            httpAllowPrivateHint: string;
            enabled: string;
            nextRunAt: string;
            createTime: string;
            trigger: string;
            triggerConfirm: string;
            triggerSuccess: string;
            log: string;
            logTitle: string;
            triggerType: string;
            triggerAuto: string;
            triggerManual: string;
            status: string;
            statusSuccess: string;
            statusFail: string;
            statusTimeout: string;
            startedAt: string;
            durationMs: string;
            errorMsg: string;
            output: string;
            noDetail: string;
            addTimer: string;
            editTimer: string;
            params: string;
            paramsPlaceholder: string;
            specPreset: string;
            specHint: string;
            deleteConfirm: string;
            toggleSuccessOn: string;
            toggleSuccessOff: string;
            enabledSearch: string;
            placeholder: {
              name: string;
              executorType: string;
              spec: string;
              methodName: string;
              httpUrl: string;
              httpBody: string;
              httpHeader: string;
            };
            form: {
              name: FormMsg;
              spec: FormMsg;
              methodName: FormMsg;
              httpUrl: FormMsg;
            };
          };
        };
        log: {
          loginlog: {
            title: string;
            listTitle: string;
            detail: string;
            userName: string;
            ipaddr: string;
            loginLocation: string;
            deviceType: string;
            browser: string;
            os: string;
            status: string;
            loginTime: string;
            accountInfo: string;
            client: string;
            msg: string;
            view: string;
            unlock: string;
            confirmUnlock: string;
            clean: string;
            confirmClean: string;
            confirmCleanButton: string;
            cleanSuccess: string;
            unlockSuccess: string;
            exportFileName: string;
            placeholder: {
              ipaddr: string;
              userName: string;
              status: string;
            };
          };
          errorlog: {
            title: string;
            listTitle: string;
            detail: string;
            form: string;
            info: string;
            level: string;
            status: string;
            solution: string;
            createTime: string;
            requestId: string;
            traceId: string;
            view: string;
            getSolution: string;
            confirmGetSolution: string;
            solutionSubmitted: string;
            placeholder: {
              form: string;
              info: string;
              level: string;
              status: string;
            };
          };
          operlog: {
            title: string;
            listTitle: string;
            detail: string;
            businessType: string;
            operName: string;
            operIp: string;
            operLocation: string;
            operId: string;
            status: string;
            operTime: string;
            costTime: string;
            operInfo: string;
            requestInfo: string;
            operParam: string;
            jsonResult: string;
            errorMsg: string;
            module: string;
            request: string;
            exportFileName: string;
            view: string;
            clean: string;
            confirmClean: string;
            confirmCleanButton: string;
            cleanSuccess: string;
            placeholder: {
              title: string;
              businessType: string;
              operName: string;
              operIp: string;
              status: string;
            };
          };
        };
      };
      form: {
        required: string;
        userName: FormMsg;
        phone: FormMsg;
        pwd: FormMsg;
        confirmPwd: FormMsg;
        code: FormMsg;
        email: FormMsg;
      };
      dropdown: Record<Global.DropdownKey, string>;
      icon: {
        themeConfig: string;
        themeSchema: string;
        lang: string;
        fullscreen: string;
        fullscreenExit: string;
        reload: string;
        collapse: string;
        expand: string;
        pin: string;
        unpin: string;
      };
      datatable: {
        itemCount: string;
        fixed: {
          left: string;
          right: string;
          unFixed: string;
        };
      };
    };

    type GetI18nKey<T extends Record<string, unknown>, K extends keyof T = keyof T> = K extends string
      ? T[K] extends Record<string, unknown>
        ? `${K}.${GetI18nKey<T[K]>}`
        : K
      : never;

    type I18nKey = GetI18nKey<Schema>;

    type TranslateOptions<Locales extends string> = import('vue-i18n').TranslateOptions<Locales>;

    interface $T {
      (key: I18nKey): string;
      (key: I18nKey, plural: number, options?: TranslateOptions<LangType>): string;
      (key: I18nKey, defaultMsg: string, options?: TranslateOptions<I18nKey>): string;
      (key: I18nKey, list: unknown[], options?: TranslateOptions<I18nKey>): string;
      (key: I18nKey, list: unknown[], plural: number): string;
      (key: I18nKey, list: unknown[], defaultMsg: string): string;
      (key: I18nKey, named: Record<string, unknown>, options?: TranslateOptions<LangType>): string;
      (key: I18nKey, named: Record<string, unknown>, plural: number): string;
      (key: I18nKey, named: Record<string, unknown>, defaultMsg: string): string;
    }
  }

  /** Service namespace */
  namespace Service {
    /** Other baseURL key */
    type OtherBaseURLKey = 'demo';

    interface ServiceConfigItem {
      /** The backend service base url */
      baseURL: string;
      /** The proxy pattern of the backend service base url */
      proxyPattern: string;
    }

    interface OtherServiceConfigItem extends ServiceConfigItem {
      key: OtherBaseURLKey;
    }

    /** The backend service config */
    interface ServiceConfig extends ServiceConfigItem {
      /** Other backend service config */
      other: OtherServiceConfigItem[];
    }

    interface SimpleServiceConfig extends Pick<ServiceConfigItem, 'baseURL'> {
      other: Record<OtherBaseURLKey, string>;
    }

    /** The backend service response data */
    type Response<T = unknown> = {
      /** The backend service response code */
      code: string;
      /** The backend service response message */
      msg: string;
      /** The backend service response data */
      data: T;
      /** The backend service response rows */
      rows?: any[];
      /** The backend service response total */
      total?: number;
    };

    /** The demo backend service response data */
    type DemoResponse<T = unknown> = {
      /** The backend service response code */
      status: string;
      /** The backend service response message */
      message: string;
      /** The backend service response data */
      result: T;
    };
  }
}
