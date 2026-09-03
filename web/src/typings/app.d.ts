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
      upgrade: {
        checkFail: string;
        alreadyLatest: string;
        startFail: string;
        foundNewVersion: string;
        startNow: string;
        startNowTip: string;
        progressTitle: string;
        stageIdle: string;
        stageDownloading: string;
        stageVerifying: string;
        stageUnpacking: string;
        stageInstalling: string;
        stageSuccess: string;
        stageFailed: string;
        stageUnreachable: string;
        successTip: string;
        unreachableTip: string;
        processingTip: string;
        background: string;
      };
      common: {
        action: string;
        add: string;
        addSuccess: string;
        backToHome: string;
        batchDelete: string;
        cancel: string;
        close: string;
        about: string;
        version: string;
        buildTime: string;
        checkUpdate: string;
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
        disable: string;
        enable: string;
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
        copyFailed: string;
        login: string;
        noChange: string;
        placeholderInput: string;
        placeholderSelect: string;
        yesOrNo: {
          yes: string;
          no: string;
        };
        iconPicker: {
          searchPlaceholder: string;
          noMatch: string;
          unset: string;
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
        server: string;
        gateway: string;
        common: string;
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
          wecomLogin: {
            title: string;
            scanTip: string;
            appName: string;
            scanned: string;
            scannedConfirm: string;
            expired: string;
            refresh: string;
            loading: string;
            qrCodeLoadFailed: string;
            countdown: string;
            backToLogin: string;
          };
        };
        init: {
          title: string;
          subTitle: string;
          noticeTitle: string;
          noticeDesc: string;
          autoTitle: string;
          autoDesc: string;
          autoSuccess: string;
          manualConfig: string;
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
          identity: {
            cardTitle: string;
            active: string;
            inactive: string;
            noIdentity: string;
            apiKeyLabel: string;
            budgetLabel: string;
            modelsLabel: string;
            expiresLabel: string;
            expiresNever: string;
            mcpLabel: string;
            skillLabel: string;
            copy: string;
            copied: string;
            budgetUnlimited: string;
            budgetModels: string;
            budgetMcps: string;
            budgetPerResource: string;
            resModel: string;
            resMcp: string;
            resSkill: string;
            resCount: string;
            resNotAuthorized: string;
            resApproval: string;
            resEmptyModels: string;
            resEmptyMarket: string;
            overviewTitle: string;
            overviewMonthBudget: string;
            overviewSpent: string;
            overviewRequests: string;
            overviewDailyAvg: string;
            overviewBudgetUsage: string;
            appsTitle: string;
            appsEmpty: string;
            typeModel: string;
            typeMcp: string;
            typeSkill: string;
            typeAgent: string;
            statusPending: string;
            statusApproved: string;
            statusRejected: string;
            appsResource: string;
            appsTime: string;
            appsNotes: string;
            goSquare: string;
            navIdentity: string;
          };
          myApps: {
            title: string;
            empty: string;
          };
          square: {
            tab: string;
            title: string;
            subtitle: string;
            searchPlaceholder: string;
            empty: string;
            authorized: string;
            notAuthorized: string;
            isNew: string;
            requiresApproval: string;
            viewAccess: string;
            copy: string;
            copied: string;
            accessTitle: string;
            accessModelKey: string;
            accessModelKeyTip: string;
            accessBaseUrl: string;
            accessApiKey: string;
            noIdentity: string;
            apply: string;
            applyPending: string;
            applyTitle: string;
            applyModel: string;
            applyReason: string;
            applyReasonPlaceholder: string;
            applySubmit: string;
            applySuccess: string;
            filterModels: string;
            filterMcps: string;
            filterSkills: string;
            searchPlaceholderSkill: string;
            emptySkills: string;
            skillDownload: string;
            skillDownloadSuccess: string;
            skillNoPackage: string;
            skillNeedApproval: string;
            accessMcpUrl: string;
            accessMcpConfig: string;
            accessMcpConfigTip: string;
            copyConfig: string;
            toolsCount: string;
          };
        };
        gateway: {
          title: string;
          comingSoon: string;
          application: {
            title: string;
            subtitle: string;
            statusAll: string;
            statusPending: string;
            statusApproved: string;
            statusRejected: string;
            typeAll: string;
            typeModel: string;
            typeMcp: string;
            typeSkill: string;
            userPlaceholder: string;
            batchApprove: string;
            batchReject: string;
            reviewNotes: string;
            reviewNotesPlaceholder: string;
            approveConfirm: string;
            rejectConfirm: string;
            batchConfirm: string;
            batchResult: string;
            reviewApprove: string;
            reviewReject: string;
            approveSuccess: string;
            rejectSuccess: string;
            reviewWarning: string;
            col: {
              applicant: string;
              resource: string;
              resourceType: string;
              reason: string;
              status: string;
              applyTime: string;
              reviewer: string;
              action: string;
            };
          };
          usage: {
            title: string;
            tabLlm: string;
            tabMcp: string;
            provider: string;
            userPlaceholder: string;
            modelPlaceholder: string;
            providerPlaceholder: string;
            unattributed: string;
            syncNow: string;
            syncConfirm: string;
            syncSuccess: string;
            reconcileNow: string;
            reconcileConfirm: string;
            reconcileSuccess: string;
            col: {
              time: string;
              user: string;
              aiKey: string;
              model: string;
              deployment: string;
              callType: string;
              promptTokens: string;
              completionTokens: string;
              cost: string;
              duration: string;
            };
          };
          mcpLog: {
            title: string;
            server: string;
            serverPlaceholder: string;
            tool: string;
            toolPlaceholder: string;
            status: string;
            statusSuccess: string;
            statusError: string;
            externalCost: string;
            internalCost: string;
            syncConfirm: string;
          };
          cost: {
            title: string;
            toLogs: string;
            preset: {
              today: string;
              yesterday: string;
              month: string;
              last7: string;
              last30: string;
            };
            search: {
              preset: string;
              dateRange: string;
              department: string;
              user: string;
              model: string;
              modelPlaceholder: string;
              provider: string;
            };
            kpi: {
              internal: string;
              external: string;
              diff: string;
              dailyAvg: string;
              change: string;
              changeTip: string;
              diffTip: string;
              dailyAvgTip: string;
            };
            trend: {
              title: string;
              internal: string;
              external: string;
            };
            top: {
              title: string;
              rank: string;
            };
            detail: {
              title: string;
              dimension: {
                department: string;
                user: string;
                model: string;
                aiKey: string;
                provider: string;
                date: string;
                mcp: string;
              };
              sort: {
                internal: string;
                external: string;
                requests: string;
                tokens: string;
              };
              col: {
                requests: string;
                promptTokens: string;
                completionTokens: string;
                totalTokens: string;
                internalCost: string;
                externalCost: string;
                costDiff: string;
                activeUsers: string;
                perCapita: string;
                operations: string;
              };
            };
          };
          adoption: {
            search: {
              preset: string;
              dateRange: string;
              department: string;
            };
            kpi: {
              coverage: string;
              coverageSubTip: string;
              coverageTip: string;
              change: string;
              newActive: string;
              newActiveTip: string;
              prevActive: string;
              dailyRequests: string;
              dailyRequestsTip: string;
              totalRequests: string;
              totalRequestsTip: string;
              perCapitaTokens: string;
              perCapitaTokensTip: string;
            };
            trend: {
              title: string;
              active: string;
              requests: string;
            };
            dept: {
              title: string;
              expandTip: string;
              col: {
                name: string;
                member: string;
                active: string;
                coverage: string;
                requests: string;
                totalTokens: string;
                internalCost: string;
              };
              user: {
                name: string;
                status: string;
                activeYes: string;
                activeNo: string;
                lastActive: string;
              };
            };
            model: {
              title: string;
              tip: string;
              col: {
                model: string;
                requestShare: string;
                costShare: string;
                activeUsers: string;
              };
            };
          };
          health: {
            checkNow: string;
            checked: string;
            checkedAt: string;
            card: {
              mcp: string;
              deployment: string;
              components: string;
              freshness: string;
              total: string;
            };
            status: {
              healthy: string;
              unhealthy: string;
              unknown: string;
            };
            freshness: {
              warning: string;
              danger: string;
              unknown: string;
              llm: string;
              mcp: string;
              lastSync: string;
              tip: string;
            };
            tab: {
              mcp: string;
              deployment: string;
              component: string;
            };
            col: {
              name: string;
              serverName: string;
              model: string;
              deployName: string;
              modelKey: string;
              status: string;
              lastCheck: string;
              error: string;
              component: string;
              latency: string;
              message: string;
            };
          };
          report: {
            listTitle: string;
            detailTitle: string;
            view: string;
            timer: string;
            copyMd: string;
            copySuccess: string;
            export: string;
            emptyDetail: string;
            type: {
              all: string;
              weekly: string;
              monthly: string;
              custom: string;
            };
            col: {
              type: string;
              period: string;
              summary: string;
              creator: string;
              operations: string;
            };
            kpi: {
              internal: string;
              external: string;
              diff: string;
              days: string;
            };
            tab: {
              dept: string;
              model: string;
              user: string;
            };
            generate: {
              title: string;
              weeklyTip: string;
              monthlyTip: string;
              rangeRequired: string;
              dateRange: string;
              confirm: string;
              success: string;
            };
          };
          mcp: {
            title: string;
            add: string;
            edit: string;
            transportSse: string;
            transportHttp: string;
            transportStdio: string;
            authNone: string;
            authApiKey: string;
            authBearer: string;
            billingFree: string;
            billingPerCall: string;
            costPerCall: string;
            internalCostPerCall: string;
            internalCostPerCallTip: string;
            healthCheck: {
              short: string;
              done: string;
              failed: string;
            };
            col: {
              name: string;
              serverName: string;
              url: string;
              transport: string;
              command: string;
              args: string;
              env: string;
              authType: string;
              authValue: string;
              category: string;
              author: string;
              iconUrl: string;
              documentationUrl: string;
              billingType: string;
              toolCount: string;
              callCount: string;
              healthStatus: string;
              isPublished: string;
              isActive: string;
              litellmSynced: string;
              description: string;
            };
            form: {
              name: { required: string };
              serverName: {
                required: string;
                pattern: string;
                tip: string;
                renameTip: string;
              };
              url: { required: string };
              command: { required: string };
              commandTip: string;
              argsPlaceholder: string;
              envKeyPlaceholder: string;
              envValuePlaceholder: string;
              envTip: string;
              authValue: string;
              authValuePlaceholder: string;
              valuesTip: string;
              costRequired: string;
              instructions: string;
              categoryPlaceholder: string;
              addToolsMissed: string;
            };
            health: {
              unknown: string;
              healthy: string;
              unhealthy: string;
            };
            publish: {
              short: string;
              title: string;
              subtitle: string;
              isPublished: string;
              autoGrantTip: string;
              visibilityType: string;
              visibilityAll: string;
              visibilitySelected: string;
              visibilityUser: string;
              visibilityMixed: string;
              departmentIds: string;
              userIds: string;
              departmentRequired: string;
              userRequired: string;
              mixedRequired: string;
              requiresApproval: string;
              requiresApprovalTip: string;
            };
            toolsDrawer: {
              short: string;
              title: string;
              tip: string;
              refresh: string;
              refreshSuccess: string;
              emptyTip: string;
              schema: string;
              schemaEmpty: string;
              editBilling: string;
              inheritServer: string;
              col: {
                toolName: string;
                displayName: string;
                description: string;
                billingType: string;
                cost: string;
              };
            };
          };
          skill: {
            title: string;
            subtitle: string;
            add: string;
            edit: string;
            actionDownload: string;
            noPackage: string;
            packageSize: string;
            upload: {
              title: string;
              tip: string;
              upload: string;
              replace: string;
              pick: string;
              current: string;
              success: string;
              createdButUploadFailed: string;
            };
            download: {
              current: string;
              success: string;
              failed: string;
            };
            col: {
              name: string;
              version: string;
              author: string;
              category: string;
              tags: string;
              iconUrl: string;
              documentationUrl: string;
              agentInstallPrompt: string;
              usageInstructions: string;
              zipPackage: string;
              zipOriginName: string;
              zipSize: string;
              installCount: string;
              isPublished: string;
              isActive: string;
              description: string;
            };
            form: {
              name: { required: string };
              version: { placeholder: string; invalid: string };
              categoryPlaceholder: string;
              tagsPlaceholder: string;
              agentInstallPromptPlaceholder: string;
              usageInstructionsPlaceholder: string;
            };
            publish: {
              short: string;
              toMarket: string;
              requiresApproval: string;
              needPackage: string;
              createdButPublishFailed: string;
            };
            usage: {
              title: string;
              tip: string;
              col: {
                userName: string;
                skillName: string;
                action: string;
                createTime: string;
              };
            };
          };
          common: {
            optional: string;
            unlimited: string;
            active: string;
            inactive: string;
            synced: string;
            unsynced: string;
            published: string;
            unpublished: string;
            createTime: string;
            billingTypeToken: string;
            billingTypePerCall: string;
            billingTypeMonthlyQuota: string;
            formatOpenai: string;
            formatAnthropic: string;
            formatLmstudio: string;
            formatOllama: string;
            categoryChat: string;
            categoryEmbedding: string;
            categoryRerank: string;
            categoryOther: string;
            keyPersonalMain: string;
            keyPersonalScene: string;
            keyDeptMain: string;
            keyDeptScene: string;
            ownerUser: string;
            ownerDept: string;
            rateLimitNone: string;
            rateLimitTotal: string;
            rateLimitPerModel: string;
            duration1d: string;
            duration7d: string;
            duration30d: string;
            dimUser: string;
            dimModel: string;
            dimAiKey: string;
            rangeToday: string;
            range7d: string;
            rangeThisMonth: string;
            range30d: string;
            rangeLastMonth: string;
            hardLimitOn: string;
            hardLimitOff: string;
          };
          provider: {
            title: string;
            add: string;
            edit: string;
            col: {
              name: string;
              providerType: string;
              isActive: string;
              description: string;
              supportedFormats: string;
            };
            form: {
              name: { required: string };
              providerType: { required: string };
            };
            selectProviderTip: string;
          };
          balance: {
            title: string;
            dashboardTitle: string;
            dashboardEmpty: string;
            vendorSideNote: string;
            sync: string;
            syncSuccess: string;
            config: string;
            configTitle: string;
            configSaved: string;
            configAccessKeyId: string;
            configAccessKeySecret: string;
            configRegion: string;
            configMaskTip: string;
            configTip: string;
            lastSync: string;
            neverSynced: string;
            typeSeat: string;
            typePackage: string;
            specStandard: string;
            specPro: string;
            specMax: string;
            col: {
              itemName: string;
              itemType: string;
              specType: string;
              status: string;
              cycleEnd: string;
              totalValue: string;
              usedValue: string;
              surplusValue: string;
              seatCount: string;
              packageCount: string;
            };
          };
          credential: {
            title: string;
            add: string;
            edit: string;
            resync: string;
            resyncSuccess: string;
            addKey: string;
            col: {
              credentialName: string;
              provider: string;
              format: string;
              litellmSynced: string;
              isActive: string;
              description: string;
              credentialValues: string;
              apiBase: string;
              apiKey: string;
            };
            form: {
              credentialName: { required: string };
              provider: { required: string };
              apiBase: { required: string };
              apiKey: { required: string };
              keyPlaceholder: string;
              valuePlaceholder: string;
              valuesTip: string;
              apiKeyPlaceholder: string;
            };
          };
          model: {
            title: string;
            add: string;
            edit: string;
            col: {
              name: string;
              modelKey: string;
              category: string;
              capabilities: string;
              deploymentCount: string;
              isPublished: string;
              isActive: string;
              description: string;
              logoProviderType: string;
            };
            form: {
              modelKey: { placeholder: string };
              name: { required: string };
              category: { required: string };
              capabilitiesPlaceholder: string;
              renameTip: string;
            };
            modelKeyUnset: string;
            selectModelTip: string;
            publish: {
              title: string;
              subtitle: string;
              isPublished: string;
              visibilityType: string;
              visibilityAll: string;
              visibilitySelected: string;
              visibilityUser: string;
              visibilityMixed: string;
              departmentIds: string;
              departmentRequired: string;
              userIds: string;
              userRequired: string;
              mixedRequired: string;
              requiresApproval: string;
              requiresApprovalTip: string;
              autoGrantTip: string;
              modelKeyUnsetTip: string;
            };
          };
          deployment: {
            manageTitle: string;
            add: string;
            edit: string;
            inlineParams: string;
            test: string;
            testing: string;
            testOk: string;
            testFail: string;
            testDetail: string;
            group: {
              billing: string;
              pricing: string;
              routing: string;
              advanced: string;
            };
            col: {
              provider: string;
              deployName: string;
              credential: string;
              billingType: string;
              costPerCall: string;
              monthlyCallQuota: string;
              externalPricing: string;
              internalPricing: string;
              inputCost: string;
              outputCost: string;
              cacheReadCost: string;
              cacheCreationCost: string;
              isActive: string;
              vendorModel: string;
              weight: string;
              order: string;
              timeout: string;
              streamTimeout: string;
              maxRetries: string;
              tags: string;
              useInPassThrough: string;
              dropParams: string;
              connectivity: string;
            };
            form: {
              deployName: { required: string };
              vendorModel: { required: string };
              credentialPlaceholder: string;
              modelKey: { required: string; tip: string };
              vendorModelTip: string;
              routingTip: string;
              passThroughTip: string;
              weightPlaceholder: string;
              orderPlaceholder: string;
              timeoutPlaceholder: string;
              streamTimeoutPlaceholder: string;
              maxRetriesPlaceholder: string;
              tagsPlaceholder: string;
              pricingTip: string;
            };
          };
          router: {
            title: string;
            desc: string;
            col: {
              routingStrategy: string;
              fallbacks: string;
              allowedFails: string;
              cooldownTime: string;
              numRetries: string;
              timeout: string;
            };
            form: {
              strategyPlaceholder: string;
              allowedFailsPlaceholder: string;
              cooldownTimePlaceholder: string;
              numRetriesPlaceholder: string;
              timeoutPlaceholder: string;
              fallbacksTip: string;
              addFallback: string;
              modelPlaceholder: string;
              fallbackModelsPlaceholder: string;
            };
            strategySimpleShuffle: string;
            strategyLatencyBased: string;
            strategyCostBased: string;
            strategyLeastBusy: string;
            strategyUsageBased: string;
          };
          aiKey: {
            title: string;
            tabKeys: string;
            tabScenario: string;
            tabKeysDesc: string;
            tabScenarioDesc: string;
            add: string;
            edit: string;
            baseSection: string;
            authSection: string;
            budgetSection: string;
            rateLimitSection: string;
            batchCreate: string;
            batchTitle: string;
            batchModeDept: string;
            batchModeUsers: string;
            batchDeptRequired: string;
            batchUsersRequired: string;
            batchSubmit: string;
            batchTip: string;
            batchResult: string;
            batchResultTotal: string;
            batchResultCreated: string;
            batchResultSkipped: string;
            batchResultFailedCount: string;
            batchResultFailed: string;
            batchResultUser: string;
            batchResultReason: string;
            batchScene: {
              title: string;
              nameTemplate: string;
              nameTemplatePlaceholder: string;
              nameTemplateTip: string;
              nameTemplateRequired: string;
              submit: string;
              tip: string;
              result: string;
            };
            rotate: string;
            rotateConfirm: string;
            rotateSuccess: string;
            batchSceneCreate: string;
            copyTemplate: string;
            resync: string;
            resyncConfirm: string;
            resyncSuccess: string;
            viewKey: string;
            hideKey: string;
            copyKey: string;
            never: string;
            expired: string;
            expiringSoon: string;
            col: {
              name: string;
              username: string;
              keyType: string;
              scenario: string;
              ownerType: string;
              ownerId: string;
              owner: string;
              keyPrefix: string;
              models: string;
              mcps: string;
              skills: string;
              budget: string;
              budgetLimit: string;
              budgetHardLimit: string;
              budgetDuration: string;
              rateLimit: string;
              rateLimitMode: string;
              tpmLimit: string;
              rpmLimit: string;
              isActive: string;
              expiresAt: string;
              lastUsedAt: string;
              description: string;
            };
            modelCount: string;
            form: {
              keyType: { required: string };
              ownerType: { required: string };
              ownerId: { required: string };
              scenarioPlaceholder: string;
              scenarioRequired: string;
              namePlaceholder: string;
              mainKeyNameFixed: string;
              modelsPlaceholder: string;
              mcpsPlaceholder: string;
              skillsPlaceholder: string;
              budgetHardLimitDesc: string;
              ownerUserPlaceholder: string;
              ownerDeptPlaceholder: string;
              expiresAtPlaceholder: string;
              keyTypeDescPersonalMain: string;
              keyTypeDescPersonalScene: string;
              keyTypeDescDeptMain: string;
              keyTypeDescDeptScene: string;
            };
          };
          keyScenario: {
            title: string;
            add: string;
            edit: string;
            col: {
              name: string;
              description: string;
            };
            form: {
              namePlaceholder: string;
              nameRequired: string;
              descPlaceholder: string;
            };
          };
          dashboard: {
            aggregate: string;
            aggregateSuccess: string;
            customRange: string;
            scopeAll: string;
            scopeSelf: string;
            totalRequests: string;
            totalCost: string;
            internalCost: string;
            totalTokens: string;
            input: string;
            output: string;
            cacheRead: string;
            budgetTotal: string;
            trendTitle: string;
            metricCost: string;
            metricRequests: string;
            metricTokens: string;
            topTitle: string;
            topName: string;
            topCost: string;
            topRequests: string;
            topTokens: string;
            budgetTitle: string;
            budgetName: string;
            budgetOwner: string;
            budgetLimit: string;
            budgetUsed: string;
            usageRate: string;
            hardLimit: string;
            isActive: string;
          };
          budget: {
            tabKey: string;
            tabDept: string;
            tabUser: string;
            add: string;
            edit: string;
            scopeType: string;
            scopeDept: string;
            scopeUser: string;
            scopeName: string;
            budgetLimit: string;
            budgetUsed: string;
            duration: string;
            duration1d: string;
            duration7d: string;
            duration30d: string;
            softWarnPercent: string;
            hardLimit: string;
            alertStatus: string;
            normal: string;
            softWarned: string;
            hardLimited: string;
            isActive: string;
            form: {
              scopeIdRequired: string;
              limitRequired: string;
            };
          };
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
            defaultRole: string;
            defaultRolePlaceholder: string;
            defaultRoleTip: string;
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
            tabNotifyWecom: string;
            tabNotifyMorning: string;
            notifyWecomPushEnabled: string;
            notifyWecomPushTip: string;
            notifyWecomRedirectBase: string;
            notifyWecomRedirectBasePlaceholder: string;
            notifyWecomMaxTargets: string;
            notifyPushBudgetAlertEnabled: string;
            notifyWecomTestUser: string;
            notifyWecomTestUserPlaceholder: string;
            notifyWecomTestBtn: string;
            notifyWecomTestSuccess: string;
            notifyWecomTestFail: string;
            notifyWecomAppSection: string;
            notifyWecomBotSection: string;
            notifyWecomBotEnabled: string;
            notifyWecomBotGroupTip: string;
            notifyWecomBotGroupAdd: string;
            notifyWecomBotGroupAddTitle: string;
            notifyWecomBotGroupName: string;
            notifyWecomBotGroupNamePlaceholder: string;
            notifyWecomBotWebhook: string;
            notifyWecomBotWebhookPlaceholder: string;
            notifyWecomBotGroupCreatedAt: string;
            notifyWecomBotGroupDeleteConfirm: string;
            notifyWecomBotTest: string;
            notifyWecomBotTestSelect: string;
            notifyWecomBotTestBtn: string;
            notifyWecomBotTestSuccess: string;
            notifyWecomBotTestFail: string;
            notifyMorningEnabled: string;
            notifyMorningSendTime: string;
            notifyMorningTargetType: string;
            notifyMorningTargetAll: string;
            notifyMorningTargetDepts: string;
            notifyMorningTargetUsers: string;
            notifyMorningTargetIds: string;
            notifyMorningChannels: string;
            notifyMorningChannelWecomApp: string;
            notifyMorningChannelWecomBot: string;
            notifyMorningChannelTip: string;
            notifyMorningBotGroups: string;
            notifyMorningTemplateTitle: string;
            notifyMorningTemplateVars: string;
            notifyMorningTemplateContent: string;
            notifyMorningTemplateMarkdown: string;
            notifyMorningTemplateReset: string;
            notifyMorningTemplateVarProviderName: string;
            notifyMorningTemplateVarUsedPercent: string;
            notifyMorningTemplateVarSurplus: string;
            notifyMorningTemplateVarTotal: string;
            notifyMorningTemplateVarResetLine: string;
            notifyMorningTemplateVarOverdrawn: string;
            notifyMorningTemplateTip: string;
            notifyMorningTip: string;
            // 认证配置
            tabAccountFunction: string;
            tabAuthWecom: string;
            tabAuthWechat: string;
            tabAuthGitee: string;
            tabAuthGithub: string;
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
            dataScope: string;
            status: string;
            remark: string;
            createTime: string;
            menuPermission: string;
            setDefaultRouter: string;
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
            module: string;
            parentId: string;
            orderNum: string;
            rootName: string;
            menuName: string;
            addMenu: string;
            addChildMenu: string;
            editMenu: string;
            perms: string;
            permsTip: string;
            apiPrefix: string;
            apiPrefixTip: string;
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
              apiPrefix: FormMsg;
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
            syncWecom: string;
            syncInProgress: string;
            syncStarted: string;
            syncTimeout: string;
            syncPollFailed: string;
            syncDone: string;
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

        userCenter: {
          personalInfo: string;
          phone: string;
          email: string;
          dept: string;
          role: string;
          createTime: string;
          baseProfile: string;
          tab: {
            profile: string;
            updatePwd: string;
            social: string;
            online: string;
          };
          profile: {
            nickName: string;
            email: string;
            phone: string;
            sex: string;
            male: string;
            female: string;
            placeholder: {
              nickName: string;
              email: string;
              phone: string;
            };
            save: string;
            updateSuccess: string;
            form: {
              nickName: string;
              sex: string;
            };
          };
          password: {
            oldPassword: string;
            newPassword: string;
            confirmPassword: string;
            placeholder: {
              oldPassword: string;
              newPassword: string;
              confirmPassword: string;
            };
            submit: string;
            modifySuccess: string;
            inconsistent: string;
            form: {
              oldPassword: string;
              newPassword: string;
              confirmPassword: string;
            };
          };
          avatar: {
            title: string;
            selectImage: string;
            confirmCrop: string;
            uploadTypeInvalid: string;
            updateSuccess: string;
          };
          social: {
            bindTime: string;
            bind: string;
            unbind: string;
            unbindSuccess: string;
            wechat: string;
            wecom: string;
            autoBindTip: string;
          };
          online: {
            deviceType: string;
            ipaddr: string;
            loginLocation: string;
            browser: string;
            os: string;
            loginTime: string;
            forceLogout: string;
            forceLogoutConfirm: string;
            forceLogoutSuccess: string;
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
