const local: App.I18n.Schema = {
  system: {
    title: 'DevOps Admin',
    updateTitle: 'System Version Update Notification',
    updateContent: 'A new version of the system has been detected. Do you want to refresh the page immediately?',
    updateConfirm: 'Refresh immediately',
    updateCancel: 'Later'
  },
  common: {
    action: 'Action',
    add: 'Add',
    addSuccess: 'Add Success',
    backToHome: 'Back to home',
    batchDelete: 'Batch Delete',
    cancel: 'Cancel',
    close: 'Close',
    check: 'Check',
    selectAll: 'Select All',
    expandColumn: 'Expand Column',
    columnSetting: 'Column Setting',
    config: 'Config',
    confirm: 'Confirm',
    delete: 'Delete',
    deleteSuccess: 'Delete Success',
    confirmDelete: 'Are you sure you want to delete?',
    edit: 'Edit',
    warning: 'Warning',
    error: 'Error',
    index: 'Index',
    keywordSearch: 'Please enter keyword',
    logout: 'Logout',
    logoutConfirm: 'Are you sure you want to log out?',
    lookForward: 'Coming soon',
    modify: 'Modify',
    modifySuccess: 'Modify Success',
    noChange: 'No changes',
    noData: 'No Data',
    operate: 'Operate',
    pleaseCheckValue: 'Please check whether the value is valid',
    refresh: 'Refresh',
    reset: 'Reset',
    search: 'Search',
    switch: 'Switch',
    tip: 'Tip',
    trigger: 'Trigger',
    save: 'Save',
    update: 'Update',
    updateSuccess: 'Update Success',
    updateExisting: 'Update Existing',
    import: 'Import',
    importSuccess: 'Import Success',
    importFail: 'Import Fail',
    importTemplate: 'Import Template',
    importTip: 'Tip: Only "xls" or "xlsx" format files are allowed',
    importSize: 'File size cannot exceed {size}',
    importFormat: 'The import data format is incorrect',
    importEnd: 'Import Completed',
    importResult: 'Successfully imported {success} items, failed {fail} items',
    downloadTemplate: 'Download Template',
    userCenter: 'User Center',
    selected: 'Selected',
    anyRecords: 'records',
    clear: 'Clear',
    noSelectRecord: 'No record selected',
    copySuccess: 'Copy successfully',
    copyNotSupported: 'Your browser does not support the Clipboard API',
    login: 'Login',
    yesOrNo: {
      yes: 'Yes',
      no: 'No'
    }
  },
  request: {
    logout: 'Logout user after request failed',
    logoutMsg: 'User status is invalid, please log in again',
    logoutWithModal: 'Pop up modal after request failed and then log out user',
    logoutWithModalMsg: 'User status is invalid, please log in again',
    refreshToken: 'The requested token has expired, refresh the token',
    tokenExpired: 'The requested token has expired'
  },
  theme: {
    themeDrawerTitle: 'Theme Configuration',
    tabs: {
      appearance: 'Appearance',
      layout: 'Layout',
      general: 'General',
      preset: 'Preset'
    },
    appearance: {
      themeSchema: {
        title: 'Theme Schema',
        light: 'Light',
        dark: 'Dark',
        auto: 'Follow System'
      },
      grayscale: 'Grayscale',
      colourWeakness: 'Colour Weakness',
      themeColor: {
        title: 'Theme Color',
        primary: 'Primary',
        info: 'Info',
        success: 'Success',
        warning: 'Warning',
        error: 'Error',
        followPrimary: 'Follow Primary'
      },
      themeRadius: {
        title: 'Theme Radius'
      },
      recommendColor: 'Apply Recommended Color Algorithm',
      recommendColorDesc: 'The recommended color algorithm refers to',
      preset: {
        title: 'Theme Presets',
        apply: 'Apply',
        applySuccess: 'Preset applied successfully',
        default: {
          name: 'Default Preset',
          desc: 'Default theme preset with balanced settings'
        },
        dark: {
          name: 'Dark Preset',
          desc: 'Dark theme preset for night time usage'
        },
        compact: {
          name: 'Compact Preset',
          desc: 'Compact layout preset for small screens'
        },
        azir: {
          name: "Azir's Preset",
          desc: 'It is a cold and elegant preset that Azir likes'
        }
      }
    },
    layout: {
      layoutMode: {
        title: 'Layout Mode',
        vertical: 'Vertical Mode',
        horizontal: 'Horizontal Mode',
        'vertical-mix': 'Vertical Mix Mode',
        'vertical-hybrid-header-first': 'Left Hybrid Header-First',
        'top-hybrid-sidebar-first': 'Top-Hybrid Sidebar-First',
        'top-hybrid-header-first': 'Top-Hybrid Header-First',
        vertical_detail: 'Vertical menu layout, with the menu on the left and content on the right.',
        'vertical-mix_detail':
          'Vertical mix-menu layout, with the primary menu on the dark left side and the secondary menu on the lighter left side.',
        'vertical-hybrid-header-first_detail':
          'Left hybrid layout, with the primary menu at the top, the secondary menu on the dark left side, and the tertiary menu on the lighter left side.',
        horizontal_detail: 'Horizontal menu layout, with the menu at the top and content below.',
        'top-hybrid-sidebar-first_detail':
          'Top hybrid layout, with the primary menu on the left and the secondary menu at the top.',
        'top-hybrid-header-first_detail':
          'Top hybrid layout, with the primary menu at the top and the secondary menu on the left.'
      },
      tab: {
        title: 'Tab Settings',
        visible: 'Tab Visible',
        cache: 'Tag Bar Info Cache',
        cacheTip: 'Keep the tab bar information after leaving the page',
        height: 'Tab Height',
        mode: {
          title: 'Tab Mode',
          slider: 'Slider',
          chrome: 'Chrome',
          button: 'Button'
        },
        closeByMiddleClick: 'Close Tab by Middle Click',
        closeByMiddleClickTip: 'Enable closing tabs by clicking with the middle mouse button'
      },
      header: {
        title: 'Header Settings',
        height: 'Header Height',
        breadcrumb: {
          visible: 'Breadcrumb Visible',
          showIcon: 'Breadcrumb Icon Visible'
        }
      },
      sider: {
        title: 'Sider Settings',
        inverted: 'Dark Sider',
        width: 'Sider Width',
        collapsedWidth: 'Sider Collapsed Width',
        mixWidth: 'Mix Sider Width',
        mixCollapsedWidth: 'Mix Sider Collapse Width',
        mixChildMenuWidth: 'Mix Child Menu Width',
        autoSelectFirstMenu: 'Auto Select First Submenu',
        autoSelectFirstMenuTip:
          'When a first-level menu is clicked, the first submenu is automatically selected and navigated to the deepest level'
      },
      footer: {
        title: 'Footer Settings',
        visible: 'Footer Visible',
        fixed: 'Fixed Footer',
        height: 'Footer Height',
        right: 'Right Footer'
      },
      content: {
        title: 'Content Area Settings',
        scrollMode: {
          title: 'Scroll Mode',
          tip: 'The theme scroll only scrolls the main part, the outer scroll can carry the header and footer together',
          wrapper: 'Wrapper',
          content: 'Content'
        },
        page: {
          animate: 'Page Animate',
          mode: {
            title: 'Page Animate Mode',
            fade: 'Fade',
            'fade-slide': 'Slide',
            'fade-bottom': 'Fade Zoom',
            'fade-scale': 'Fade Scale',
            'zoom-fade': 'Zoom Fade',
            'zoom-out': 'Zoom Out',
            none: 'None'
          }
        },
        fixedHeaderAndTab: 'Fixed Header And Tab'
      }
    },
    general: {
      title: 'General Settings',
      watermark: {
        title: 'Watermark Settings',
        visible: 'Watermark Full Screen Visible',
        text: 'Custom Watermark Text',
        enableUserName: 'Enable User Name Watermark',
        enableTime: 'Show Current Time',
        timeFormat: 'Time Format'
      },
      multilingual: {
        title: 'Multilingual Settings',
        visible: 'Display multilingual button'
      },
      globalSearch: {
        title: 'Global Search Settings',
        visible: 'Display GlobalSearch button'
      }
    },
    configOperation: {
      copyConfig: 'Copy Config',
      copySuccessMsg: 'Copy Success, Please replace the variable "themeSettings" in "src/theme/settings.ts"',
      resetConfig: 'Reset Config',
      resetSuccessMsg: 'Reset Success'
    }
  },
  route: {
    login: 'Login',
    403: 'No Permission',
    404: 'Page Not Found',
    500: 'Server Error',
    'iframe-page': 'Iframe',
    admin: 'Home',
    'social-callback': 'Social Callback',
    'user-center': 'User Center',
    system: 'System',
    system_user: 'User',
    system_role: 'Role',
    system_menu: 'Menu',
    system_dept: 'Department',
    system_post: 'Post',
    system_dict: 'Dictionary',
    system_notice: 'Notice',
    system_setting: 'System Setting',
    log: 'Log',
    log_loginlog: 'Login Log',
    disk: 'Disk',
    server: 'Server',
    gateway: 'AI Gateway',
    init: 'System Init'
  },
  page: {
    login: {
      common: {
        title: 'Welcome',
        subTitle: 'Please enter your account information to continue',
        loginOrRegister: 'Login / Register',
        userNamePlaceholder: 'Please enter user name',
        phonePlaceholder: 'Please enter phone number',
        codePlaceholder: 'Please enter verification code',
        passwordPlaceholder: 'Please enter password',
        confirmPasswordPlaceholder: 'Please enter password again',
        codeLogin: 'Verification code login',
        confirm: 'Confirm',
        back: 'Back',
        validateSuccess: 'Verification passed',
        loginSuccess: 'Login successfully',
        welcomeBack: 'Welcome back, {userName} !',
        register: 'Register'
      },
      pwdLogin: {
        title: 'Password Login',
        rememberMe: 'Remember me',
        forgetPassword: 'Forget password?',
        register: 'Register',
        otherAccountLogin: 'Other Account Login',
        otherLoginMode: 'Other Login Mode',
        superAdmin: 'Super Admin',
        admin: 'Admin',
        user: 'User'
      },
      codeLogin: {
        title: 'Verification Code Login',
        getCode: 'Get verification code',
        reGetCode: 'Reacquire after {time}s',
        sendCodeSuccess: 'Verification code sent successfully',
        imageCodePlaceholder: 'Please enter image verification code'
      },
      register: {
        title: 'Register',
        agreement: 'I have read and agree to',
        protocol: '《User Agreement》',
        policy: '《Privacy Policy》'
      },
      resetPwd: {
        title: 'Reset Password'
      },
      bindWeChat: {
        title: 'Bind WeChat'
      },
      captcha: {
        title: 'Security Verification',
        clickTitle: 'Click the text in order',
        slideTitle: 'Drag the slider to complete the puzzle',
        rotateTitle: 'Rotate the image to the correct direction',
        loginWithCaptcha: 'Verify and Login'
      }
    },
    init: {
      title: 'System Initialization',
      subTitle: 'Complete the database initialization before first use',
      noticeTitle: 'Before You Start',
      noticeDesc:
        'Initialization will create the database, auto-migrate tables and seed base data (roles, menus, admin account). Ensure the database service is reachable; use innoDB for MySQL.',
      confirm: 'I understand, start configuration',
      back: 'Back to login',
      form: {
        adminPassword: 'Admin Password',
        adminPasswordPlaceholder: 'Initial admin password (at least 6 characters)',
        dbType: 'Database Type',
        host: 'Host',
        hostPlaceholder: 'Enter database host',
        port: 'Port',
        portPlaceholder: 'Enter database port',
        userName: 'Username',
        userNamePlaceholder: 'Enter database username',
        password: 'Password',
        passwordPlaceholder: 'Enter database password',
        dbName: 'Database Name',
        dbNamePlaceholder: 'Enter database name',
        dbPath: 'Database File Path',
        dbPathPlaceholder: 'Enter sqlite file path',
        template: 'PG Template',
        templatePlaceholder: 'Enter postgresql template',
        redisAddr: 'Redis Address',
        redisAddrPlaceholder: 'e.g. 127.0.0.1:6379',
        redisPassword: 'Redis Password',
        redisPasswordPlaceholder: 'Leave empty if none',
        redisDB: 'Redis DB',
        redisDBPlaceholder: 'e.g. 0'
      },
      step: {
        db: 'Database',
        redis: 'Redis',
        admin: 'Admin Password'
      },
      rule: {
        adminPasswordLength: 'Admin password must be at least 6 characters',
        redisAddrRequired: 'Redis address is required'
      },
      testConnection: 'Test Connection',
      testing: 'Testing…',
      testConnectionSuccess: 'Connected',
      testConnectionFailed: 'Connection failed',
      next: 'Next',
      prev: 'Previous',
      finish: 'Finish',
      submit: 'Initialize Now',
      submitting: 'Initializing database, please wait…',
      successTitle: 'Initialization Completed',
      toLogin: 'Go to Login'
    },
    home: {
      greeting: 'Good morning, {userName}, today is another day full of vitality!',
      weatherDesc: 'Today is cloudy to clear, 20℃ - 25℃!',
      projectCount: 'Project Count',
      todo: 'Todo',
      message: 'Message',
      downloadCount: 'Download Count',
      registerCount: 'Register Count',
      schedule: 'Work and rest Schedule',
      study: 'Study',
      work: 'Work',
      rest: 'Rest',
      entertainment: 'Entertainment',
      visitCount: 'Visit Count',
      turnover: 'Turnover',
      dealCount: 'Deal Count',
      projectNews: {
        title: 'Project News',
        moreNews: 'More News',
        desc1: 'Soybean created the open source project soybean-admin on May 28, 2021!',
        desc2: 'Yanbowe submitted a bug to soybean-admin, the multi-tab bar will not adapt.',
        desc3: 'Soybean is ready to do sufficient preparation for the release of soybean-admin!',
        desc4: 'Soybean is busy writing project documentation for soybean-admin!',
        desc5: 'Soybean just wrote some of the workbench pages casually, and it was enough to see!'
      },
      creativity: 'Creativity'
    },
    system: {
      user: {
        title: 'User Management',
        userName: 'User Name',
        nickName: 'Nick Name',
        sex: 'Gender',
        roleIds: 'Role',
        postIds: 'Post',
        deptName: 'Department',
        email: 'Email',
        phonenumber: 'Phone Number',
        status: 'Status',
        avatar: 'Avatar',
        remark: 'Remark',
        createTime: 'Create Time',
        password: 'Password',
        confirmPassword: 'Confirm Password',
        statusChangeSuccess: 'Status changed successfully',
        addUser: 'Add User',
        editUser: 'Edit User',
        form: {
          userName: { required: 'Please enter user name', invalid: 'User name format is incorrect' },
          nickName: { required: 'Please enter nick name', invalid: 'Nick name format is incorrect' },
          deptId: { required: 'Please select department', invalid: 'Please select department' },
          phonenumber: { required: 'Please enter phone number', invalid: 'Phone number format is incorrect' },
          status: { required: 'Please select status', invalid: 'Please select status' },
          password: { required: 'Please enter password', invalid: 'Password format is incorrect' },
          confirmPassword: { required: 'Please enter password again', invalid: 'The two passwords are inconsistent' },
          sex: { required: 'Please select gender', invalid: 'Please select gender' },
          email: { required: 'Please enter email', invalid: 'Email format is incorrect' },
          roleIds: { required: 'Please select role', invalid: 'Please select role' },
          postIds: { required: 'Please select post', invalid: 'Please select post' },
          remark: { required: 'Please enter remark', invalid: 'Remark format is incorrect' }
        }
      },
      role: {
        title: 'Role Management',
        listTitle: 'Role List',
        roleName: 'Role Name',
        roleKey: 'Role Permission String',
        roleKeyShort: 'Permission Key',
        roleSort: 'Display Order',
        status: 'Role Status',
        remark: 'Remark',
        createTime: 'Create Time',
        menuPermission: 'Menu Permission',
        addRole: 'Add Role',
        editRole: 'Edit Role',
        assignUser: 'Assign User',
        assignUserTitle: 'Assign User Permission',
        statusChangeSuccess: 'Status changed successfully',
        roleKeyTip: "Permission key defined in the controller, e.g.: @SaCheckRole('admin')",
        form: {
          roleId: { required: 'Role ID cannot be empty', invalid: 'Role ID cannot be empty' },
          roleName: { required: 'Please enter the role name', invalid: 'Role name format is incorrect' },
          roleKey: { required: 'Please enter the permission key', invalid: 'Permission key format is incorrect' },
          roleSort: { required: 'Please enter the display order', invalid: 'Display order format is incorrect' },
          status: { required: 'Please select the role status', invalid: 'Please select the role status' },
          remark: { required: 'Please enter a remark', invalid: 'Remark format is incorrect' }
        }
      },
      menu: {
        title: 'Menu Management',
        parentId: 'Parent Menu',
        orderNum: 'Display Order',
        rootName: 'Root Category',
        menuName: 'Menu Name',
        addMenu: 'Add Menu',
        addChildMenu: 'Add Submenu',
        editMenu: 'Edit Menu',
        perms: 'Permission Key',
        permsTip: 'Permission key defined in the controller, e.g.: system:user:list',
        status: 'Menu Status',
        statusTip: 'If disabled, this menu and its submenus will be invisible',
        emptyMenu: 'No menu data',
        menuDetail: 'Menu Detail',
        menuType: 'Menu Type',
        component: 'Component Path',
        componentTip: 'Component path under the views directory, e.g.: system/user/index',
        layout: 'Layout',
        layoutTip: 'When blank layout is selected, the menu opens in a new page',
        icon: 'Icon',
        iconType: 'Icon Type',
        iconifyTip: 'Supports Iconify icons and local SVG icons',
        path: 'Route Path',
        pathTip: 'Route address to access, e.g.: user',
        externalPath: 'External URL',
        query: 'Route Params',
        iframeQuery: 'Iframe URL',
        isFrame: 'External Link',
        isFrameTip: 'Select Yes for an external link route, or Iframe to embed a web page',
        visible: 'Visible',
        visibleTip: 'Hidden menus are still accessible, just not shown in the sidebar',
        isCache: 'Cache',
        cache: 'Cached',
        isCacheTip: 'When cached, keep-alive preserves state on page refresh',
        noCache: 'Not Cached',
        cascadeDelete: 'Cascade Delete',
        cascadeDeleteContent: 'Cascade delete the selected menus and all their submenus?',
        createTime: 'Create Time',
        buttonPermissionList: 'Button Permission List',
        placeholder: {
          localIconPlaceholder: 'Please select a local icon',
          iconifyIconPlaceholder: 'Please enter the Iconify icon name',
          queryKey: 'Key cannot be empty',
          queryValue: 'Value cannot be empty',
          queryIframe: 'Please enter the iframe URL'
        },
        form: {
          parentId: { required: 'Please select the parent menu', invalid: 'Please select the parent menu' },
          menuIds: { required: 'Please select the menus to delete', invalid: 'Please select the menus to delete' },
          menuName: { required: 'Please enter the menu name', invalid: 'Menu name format is incorrect' },
          orderNum: { required: 'Please enter the display order', invalid: 'Display order format is incorrect' },
          perms: { required: 'Please enter the permission key', invalid: 'Permission key format is incorrect' },
          path: { required: 'Please enter the route path', invalid: 'Route path format is incorrect' },
          component: { required: 'Please enter the component path', invalid: 'Component path format is incorrect' }
        }
      },
      dept: {
        title: 'Department',
        parentId: 'Parent Dept',
        deptName: 'Dept Name',
        deptCategory: 'Dept Category',
        leader: 'Leader',
        phone: 'Phone',
        email: 'Email',
        sort: 'Sort',
        status: 'Status',
        createTime: 'Create Time',
        orderNum: 'Display Order',
        expandAll: 'Expand All',
        collapseAll: 'Collapse All',
        empty: 'No department data',
        addDept: 'Add Department',
        editDept: 'Edit Department',
        placeholder: {
          defaultLeaderPlaceHolder: 'Please select a leader',
          addDataLeaderPlaceHolder: 'Leader cannot be selected when adding a department',
          deptUserIsEmptyLeaderPlaceHolder: 'No users in this department, leader cannot be selected'
        },
        error: {
          getDeptDataFail: 'Failed to load department data',
          getDeptUserDataFail: 'Failed to load department users'
        },
        form: {
          deptId: { required: 'Please enter department id', invalid: 'Department id cannot be empty' },
          parentId: { required: 'Please select parent department', invalid: 'Please select parent department' },
          orderNum: { required: 'Please enter display order', invalid: 'Display order cannot be empty' },
          deptName: { required: 'Please enter department name', invalid: 'Department name cannot be empty' },
          deptCategory: { required: 'Please enter department category', invalid: 'Invalid department category' },
          status: { required: 'Please select status', invalid: 'Please select status' },
          phone: { required: 'Please enter phone number', invalid: 'Invalid phone number' },
          email: { required: 'Please enter email', invalid: 'Invalid email' }
        }
      },
      post: {
        title: 'Post Management',
        listTitle: 'Post List',
        deptTreeTitle: 'Departments',
        emptyDept: 'No department data',
        exportFileName: 'Post Info',
        belongDept: 'Belong Dept',
        postCode: 'Post Code',
        postCategory: 'Category Code',
        postName: 'Post Name',
        postSort: 'Display Order',
        status: 'Status',
        remark: 'Remark',
        createTime: 'Create Time',
        addPost: 'Add Post',
        editPost: 'Edit Post',
        form: {
          postId: { required: 'Please enter post id', invalid: 'Post id cannot be empty' },
          deptId: { required: 'Please select belong department', invalid: 'Belong department cannot be empty' },
          postCode: { required: 'Please enter post code', invalid: 'Post code cannot be empty' },
          postCategory: { required: 'Please enter category code', invalid: 'Invalid category code' },
          postName: { required: 'Please enter post name', invalid: 'Post name cannot be empty' },
          postSort: { required: 'Please enter display order', invalid: 'Display order cannot be empty' },
          status: { required: 'Please select status', invalid: 'Status cannot be empty' },
          remark: { required: 'Please enter remark', invalid: 'Invalid remark' }
        }
      },
      dict: {
        title: 'Dictionary',
        dictName: 'Dict Name',
        dictTypeTitle: 'Dict Type',
        dictType: 'Dict Type',
        dictData: 'Dict Data',
        addDictType: 'Add Dict Type',
        addDictData: 'Add Dict Data',
        exportDictType: 'Export Dict Type',
        editDictData: 'Edit Dict Data',
        editDictType: 'Edit Dict Type',
        refreshDictType: 'Refresh Dict',
        dictTypeIsEmpty: 'Please select a dict type first',
        refreshCache: 'Refresh Cache',
        confirmDeleteDictType: 'Confirm delete dict type',
        refreshCacheSuccess: 'Cache refreshed successfully',
        remark: 'Remark',
        data: {
          label: 'Dict Label',
          value: 'Dict Value',
          dictSort: 'Dict Sort',
          listClass: 'List Class',
          cssClass: 'CSS Class',
          remark: 'Remark',
          createTime: 'Create Time',
          isDefault: 'Is Default'
        },
        form: {
          dictName: { required: 'Please enter dict name', invalid: 'Dict name cannot be empty' },
          dictCode: { required: 'Please enter dict code', invalid: 'Dict code cannot be empty' },
          dictLabel: { required: 'Please enter dict label', invalid: 'Dict label cannot be empty' },
          dictValue: { required: 'Please enter dict value', invalid: 'Dict value cannot be empty' },
          dictType: { required: 'Please enter dict type', invalid: 'Dict type cannot be empty' },
          listClass: { required: 'Please select list class', invalid: 'Please select list class' },
          cssClass: { required: 'Please enter CSS class', invalid: 'Invalid CSS class' },
          dictSort: { required: 'Please enter dict sort', invalid: 'Dict sort cannot be empty' },
          remark: { required: 'Please enter remark', invalid: 'Invalid remark' }
        }
      }
    }
  },
  form: {
    required: 'Cannot be empty',
    userName: {
      required: 'Please enter user name',
      invalid: 'User name format is incorrect'
    },
    phone: {
      required: 'Please enter phone number',
      invalid: 'Phone number format is incorrect'
    },
    pwd: {
      required: 'Please enter password',
      invalid: '6-18 characters, including letters, numbers, and underscores'
    },
    confirmPwd: {
      required: 'Please enter password again',
      invalid: 'The two passwords are inconsistent'
    },
    code: {
      required: 'Please enter verification code',
      invalid: 'Verification code format is incorrect'
    },
    email: {
      required: 'Please enter email',
      invalid: 'Email format is incorrect'
    }
  },
  dropdown: {
    closeCurrent: 'Close Current',
    closeOther: 'Close Other',
    closeLeft: 'Close Left',
    closeRight: 'Close Right',
    closeAll: 'Close All',
    pin: 'Pin Tab',
    unpin: 'Unpin Tab'
  },
  icon: {
    themeConfig: 'Theme Configuration',
    themeSchema: 'Theme Schema',
    lang: 'Switch Language',
    fullscreen: 'Fullscreen',
    fullscreenExit: 'Exit Fullscreen',
    reload: 'Reload Page',
    collapse: 'Collapse Menu',
    expand: 'Expand Menu',
    pin: 'Pin',
    unpin: 'Unpin'
  },
  datatable: {
    itemCount: 'Total {total} items',
    fixed: {
      left: 'Left Fixed',
      right: 'Right Fixed',
      unFixed: 'Unfixed'
    }
  }
};

export default local;
