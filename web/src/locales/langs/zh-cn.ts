const local: App.I18n.Schema = {
  system: {
    title: 'DevOps 管理后台',
    updateTitle: '系统版本更新通知',
    updateContent: '检测到系统有新版本发布，是否立即刷新页面？',
    updateConfirm: '立即刷新',
    updateCancel: '稍后再说'
  },
  common: {
    action: '操作',
    add: '新增',
    addSuccess: '添加成功',
    backToHome: '返回首页',
    batchDelete: '批量删除',
    cancel: '取消',
    close: '关闭',
    check: '勾选',
    selectAll: '全选',
    expandColumn: '展开列',
    columnSetting: '列设置',
    config: '配置',
    confirm: '确认',
    delete: '删除',
    deleteSuccess: '删除成功',
    confirmDelete: '确认删除吗？',
    edit: '编辑',
    warning: '警告',
    error: '错误',
    index: '序号',
    keywordSearch: '请输入关键词搜索',
    logout: '退出登录',
    logoutConfirm: '确认退出登录吗？',
    lookForward: '敬请期待',
    modify: '修改',
    modifySuccess: '修改成功',
    noChange: '数据未发生变化',
    noData: '无数据',
    operate: '操作',
    pleaseCheckValue: '请检查输入的值是否合法',
    refresh: '刷新',
    reset: '重置',
    search: '搜索',
    switch: '切换',
    tip: '提示',
    trigger: '触发',
    save: '保存',
    update: '更新',
    updateSuccess: '更新成功',
    updateExisting: '更新已存在的数据',
    import: '导入',
    importSuccess: '导入成功',
    importFail: '导入失败',
    importTemplate: '导入模板',
    importTip: '提示：仅允许导入 "xls" 或 "xlsx" 格式文件',
    importSize: '文件大小不能超过 {size}',
    importFormat: '导入数据格式不正确',
    importEnd: '导入完成',
    importResult: '成功导入 {success} 条，失败 {fail} 条',
    downloadTemplate: '下载模板',
    userCenter: '个人中心',
    selected: '已选择',
    anyRecords: '条',
    clear: '清空',
    noSelectRecord: '未选择任何记录',
    copySuccess: '复制成功',
    copyNotSupported: '您的浏览器不支持 Clipboard API',
    login: '登录',
    yesOrNo: {
      yes: '是',
      no: '否'
    }
  },
  request: {
    logout: '请求失败后登出用户',
    logoutMsg: '用户状态失效，请重新登录',
    logoutWithModal: '请求失败后弹出模态框再登出用户',
    logoutWithModalMsg: '用户状态失效，请重新登录',
    refreshToken: '请求的token已过期，刷新token',
    tokenExpired: 'token已过期'
  },
  theme: {
    themeDrawerTitle: '主题配置',
    tabs: {
      appearance: '外观',
      layout: '布局',
      general: '通用',
      preset: '预设'
    },
    appearance: {
      themeSchema: {
        title: '主题模式',
        light: '亮色模式',
        dark: '暗黑模式',
        auto: '跟随系统'
      },
      grayscale: '灰色模式',
      colourWeakness: '色弱模式',
      themeColor: {
        title: '主题颜色',
        primary: '主色',
        info: '信息色',
        success: '成功色',
        warning: '警告色',
        error: '错误色',
        followPrimary: '跟随主色'
      },
      themeRadius: {
        title: '主题圆角'
      },
      recommendColor: '应用推荐算法的颜色',
      recommendColorDesc: '推荐颜色的算法参照',
      preset: {
        title: '主题预设',
        apply: '应用',
        applySuccess: '预设应用成功',
        default: {
          name: '默认预设',
          desc: 'Soybean 默认主题预设'
        },
        dark: {
          name: '暗色预设',
          desc: '适用于夜间使用的暗色主题预设'
        },
        compact: {
          name: '紧凑型',
          desc: '适用于小屏幕的紧凑布局预设'
        },
        azir: {
          name: 'Azir的预设',
          desc: '是 Azir 比较喜欢的莫兰迪色系冷淡风'
        }
      }
    },
    layout: {
      layoutMode: {
        title: '布局模式',
        vertical: '左侧菜单模式',
        'vertical-mix': '左侧菜单混合模式',
        'vertical-hybrid-header-first': '左侧混合-顶部优先',
        horizontal: '顶部菜单模式',
        'top-hybrid-sidebar-first': '顶部混合-侧边优先',
        'top-hybrid-header-first': '顶部混合-顶部优先',
        vertical_detail: '左侧菜单布局，菜单在左，内容在右。',
        'vertical-mix_detail': '左侧双菜单布局，一级菜单在左侧深色区域，二级菜单在左侧浅色区域。',
        'vertical-hybrid-header-first_detail':
          '左侧混合布局，一级菜单在顶部，二级菜单在左侧深色区域，三级菜单在左侧浅色区域。',
        horizontal_detail: '顶部菜单布局，菜单在顶部，内容在下方。',
        'top-hybrid-sidebar-first_detail': '顶部混合布局，一级菜单在左侧，二级菜单在顶部。',
        'top-hybrid-header-first_detail': '顶部混合布局，一级菜单在顶部，二级菜单在左侧。'
      },
      tab: {
        title: '标签栏设置',
        visible: '显示标签栏',
        cache: '标签栏信息缓存',
        cacheTip: '离开页面后仍然保留标签栏信息',
        height: '标签栏高度',
        mode: {
          title: '标签栏风格',
          slider: '滑块风格',
          chrome: '谷歌风格',
          button: '按钮风格'
        },
        closeByMiddleClick: '鼠标中键关闭标签页',
        closeByMiddleClickTip: '启用后可以使用鼠标中键点击标签页进行关闭'
      },
      header: {
        title: '头部设置',
        height: '头部高度',
        breadcrumb: {
          visible: '显示面包屑',
          showIcon: '显示面包屑图标'
        }
      },
      sider: {
        title: '侧边栏设置',
        inverted: '深色侧边栏',
        width: '侧边栏宽度',
        collapsedWidth: '侧边栏折叠宽度',
        mixWidth: '混合布局侧边栏宽度',
        mixCollapsedWidth: '混合布局侧边栏折叠宽度',
        mixChildMenuWidth: '混合布局子菜单宽度',
        autoSelectFirstMenu: '自动选择第一个子菜单',
        autoSelectFirstMenuTip: '点击一级菜单时，自动选择并导航到第一个子菜单的最深层级'
      },
      footer: {
        title: '底部设置',
        visible: '显示底部',
        fixed: '固定底部',
        height: '底部高度',
        right: '底部居右'
      },
      content: {
        title: '内容区域设置',
        scrollMode: {
          title: '滚动模式',
          tip: '主题滚动仅 main 部分滚动，外层滚动可携带头部底部一起滚动',
          wrapper: '外层滚动',
          content: '主体滚动'
        },
        page: {
          animate: '页面切换动画',
          mode: {
            title: '页面切换动画类型',
            'fade-slide': '滑动',
            fade: '淡入淡出',
            'fade-bottom': '底部消退',
            'fade-scale': '缩放消退',
            'zoom-fade': '渐变',
            'zoom-out': '闪现',
            none: '无'
          }
        },
        fixedHeaderAndTab: '固定头部和标签栏'
      }
    },
    general: {
      title: '通用设置',
      watermark: {
        title: '水印设置',
        visible: '显示全屏水印',
        text: '自定义水印文本',
        enableUserName: '启用用户名水印',
        enableTime: '显示当前时间',
        timeFormat: '时间格式'
      },
      multilingual: {
        title: '多语言设置',
        visible: '显示多语言按钮'
      },
      globalSearch: {
        title: '全局搜索设置',
        visible: '显示全局搜索按钮'
      }
    },
    configOperation: {
      copyConfig: '复制配置',
      copySuccessMsg: '复制成功，请替换 src/theme/settings.ts 中的变量 themeSettings',
      resetConfig: '重置配置',
      resetSuccessMsg: '重置成功'
    }
  },
  route: {
    login: '登录',
    403: '无权限',
    404: '页面不存在',
    500: '服务器错误',
    'iframe-page': '外链页面',
    admin: '首页',
    'social-callback': '社交回调',
    'user-center': '用户中心',
    system: '系统管理',
    system_user: '用户管理',
    system_role: '角色管理',
    system_menu: '菜单管理',
    system_dept: '部门管理',
    system_post: '岗位管理',
    system_dict: '字典管理',
    system_notice: '通知公告',
    system_setting: '系统设置',
    log: '日志中心',
    log_loginlog: '登录日志',
    disk: '网盘',
    server: '服务器管理',
    gateway: 'AI 网关',
    init: '系统初始化'
  },
  page: {
    login: {
      common: {
        title: '欢迎登录',
        subTitle: '请输入您的账户信息以继续',
        loginOrRegister: '登录 / 注册',
        userNamePlaceholder: '请输入用户名',
        phonePlaceholder: '请输入手机号',
        codePlaceholder: '请输入验证码',
        passwordPlaceholder: '请输入密码',
        confirmPasswordPlaceholder: '请再次输入密码',
        codeLogin: '验证码登录',
        confirm: '确定',
        back: '返回',
        validateSuccess: '验证成功',
        loginSuccess: '登录成功',
        welcomeBack: '欢迎回来，{userName} ！',
        register: '注册'
      },
      pwdLogin: {
        title: '密码登录',
        rememberMe: '记住我',
        forgetPassword: '忘记密码？',
        register: '注册账号',
        otherAccountLogin: '其他账号登录',
        otherLoginMode: '其他登录方式',
        superAdmin: '超级管理员',
        admin: '管理员',
        user: '普通用户'
      },
      codeLogin: {
        title: '验证码登录',
        getCode: '获取验证码',
        reGetCode: '{time}秒后重新获取',
        sendCodeSuccess: '验证码发送成功',
        imageCodePlaceholder: '请输入图片验证码'
      },
      register: {
        title: '注册账号',
        agreement: '我已经仔细阅读并接受',
        protocol: '《用户协议》',
        policy: '《隐私权政策》'
      },
      resetPwd: {
        title: '重置密码'
      },
      bindWeChat: {
        title: '绑定微信'
      }
    },
    init: {
      title: '系统初始化',
      subTitle: '首次使用前，请完成数据库初始化配置',
      noticeTitle: '初始化须知',
      noticeDesc:
        '初始化将创建数据库、自动建表并写入基础数据（角色、菜单、管理员账号等）。请确保数据库服务可用；MySQL 请使用 innoDB 引擎。',
      confirm: '我已确认，开始配置',
      back: '返回登录',
      form: {
        adminPassword: '管理员密码',
        adminPasswordPlaceholder: '初始 admin 账号密码（不少于 6 位）',
        dbType: '数据库类型',
        host: '数据库地址',
        hostPlaceholder: '请输入数据库地址',
        port: '数据库端口',
        portPlaceholder: '请输入数据库端口',
        userName: '用户名',
        userNamePlaceholder: '请输入数据库用户名',
        password: '密码',
        passwordPlaceholder: '请输入数据库密码',
        dbName: '数据库名',
        dbNamePlaceholder: '请输入数据库名',
        dbPath: '数据库文件路径',
        dbPathPlaceholder: '请输入 sqlite 文件存放路径',
        template: 'PG 模板',
        templatePlaceholder: '请输入 postgresql 模板',
        redisAddr: 'Redis 地址',
        redisAddrPlaceholder: '请输入 Redis 地址，如 127.0.0.1:6379',
        redisPassword: 'Redis 密码',
        redisPasswordPlaceholder: '无密码可留空',
        redisDB: 'Redis 库号',
        redisDBPlaceholder: '请输入库号，如 0'
      },
      step: {
        db: '数据库',
        redis: 'Redis',
        admin: '管理员密码'
      },
      rule: {
        adminPasswordLength: '管理员密码长度不能小于 6 位',
        redisAddrRequired: '请输入 Redis 地址'
      },
      testConnection: '测试连接',
      testing: '测试中…',
      testConnectionSuccess: '连接成功',
      testConnectionFailed: '连接失败',
      next: '下一步',
      prev: '上一步',
      finish: '完成',
      submit: '立即初始化',
      submitting: '正在初始化数据库，请稍候…',
      successTitle: '初始化完成',
      toLogin: '前往登录'
    },
    home: {
      greeting: '早安，{userName}, 今天又是充满活力的一天!',
      weatherDesc: '今日多云转晴，20℃ - 25℃!',
      projectCount: '项目数',
      todo: '待办',
      message: '消息',
      downloadCount: '下载量',
      registerCount: '注册量',
      schedule: '作息安排',
      study: '学习',
      work: '工作',
      rest: '休息',
      entertainment: '娱乐',
      visitCount: '访问量',
      turnover: '成交额',
      dealCount: '成交量',
      projectNews: {
        title: '项目动态',
        moreNews: '更多动态',
        desc1: 'Soybean 在2021年5月28日创建了开源项目 soybean-admin!',
        desc2: 'Yanbowe 向 soybean-admin 提交了一个bug，多标签栏不会自适应。',
        desc3: 'Soybean 准备为 soybean-admin 的发布做充分的准备工作!',
        desc4: 'Soybean 正在忙于为soybean-admin写项目说明文档！',
        desc5: 'Soybean 刚才把工作台页面随便写了一些，凑合能看了！'
      },
      creativity: '创意'
    },
    system: {
      user: {
        title: '用户管理',
        userName: '用户名称',
        nickName: '用户昵称',
        sex: '性别',
        roleIds: '角色',
        postIds: '岗位',
        deptName: '部门',
        email: '邮箱',
        phonenumber: '手机号码',
        status: '状态',
        avatar: '头像',
        remark: '备注',
        createTime: '创建时间',
        password: '密码',
        confirmPassword: '确认密码',
        statusChangeSuccess: '状态修改成功',
        addUser: '新增用户',
        editUser: '编辑用户',
        form: {
          userName: { required: '请输入用户名称', invalid: '用户名称格式不正确' },
          nickName: { required: '请输入用户昵称', invalid: '用户昵称格式不正确' },
          deptId: { required: '请选择归属部门', invalid: '请选择归属部门' },
          phonenumber: { required: '请输入手机号码', invalid: '手机号码格式不正确' },
          status: { required: '请选择状态', invalid: '请选择状态' },
          password: { required: '请输入密码', invalid: '密码格式不正确' },
          confirmPassword: { required: '请再次输入密码', invalid: '两次输入密码不一致' },
          sex: { required: '请选择性别', invalid: '请选择性别' },
          email: { required: '请输入邮箱', invalid: '邮箱格式不正确' },
          roleIds: { required: '请选择角色', invalid: '请选择角色' },
          postIds: { required: '请选择岗位', invalid: '请选择岗位' },
          remark: { required: '请输入备注', invalid: '备注格式不正确' }
        }
      },
      role: {
        title: '角色管理',
        listTitle: '角色列表',
        roleName: '角色名称',
        roleKey: '角色权限字符串',
        roleKeyShort: '权限字符',
        roleSort: '显示顺序',
        status: '角色状态',
        remark: '备注',
        createTime: '创建时间',
        menuPermission: '菜单权限',
        addRole: '新增角色',
        editRole: '编辑角色',
        assignUser: '分配用户',
        assignUserTitle: '分配用户权限',
        statusChangeSuccess: '状态修改成功',
        roleKeyTip: "控制器中定义的权限字符，如：@SaCheckRole('admin')",
        form: {
          roleId: { required: '角色ID不能为空', invalid: '角色ID不能为空' },
          roleName: { required: '请输入角色名称', invalid: '角色名称格式不正确' },
          roleKey: { required: '请输入权限字符', invalid: '权限字符格式不正确' },
          roleSort: { required: '请输入显示顺序', invalid: '显示顺序格式不正确' },
          status: { required: '请选择角色状态', invalid: '请选择角色状态' },
          remark: { required: '请输入备注', invalid: '备注格式不正确' }
        }
      },
      menu: {
        title: '菜单管理',
        parentId: '上级菜单',
        orderNum: '显示顺序',
        rootName: '主类目',
        menuName: '菜单名称',
        addMenu: '新增菜单',
        addChildMenu: '新增子菜单',
        editMenu: '编辑菜单',
        perms: '权限标识',
        permsTip: '控制器中定义的权限字符，如：system:user:list',
        status: '菜单状态',
        statusTip: '选择停用则该菜单及其子菜单都不可见',
        emptyMenu: '暂无菜单数据',
        menuDetail: '菜单详情',
        menuType: '菜单类型',
        component: '组件路径',
        componentTip: 'views 目录下的组件路径，如：system/user/index',
        layout: '布局',
        layoutTip: '选择空白布局时，菜单将在新页面打开',
        icon: '图标',
        iconType: '图标类型',
        iconifyTip: '支持 Iconify 图标与本地 SVG 图标',
        path: '路由地址',
        pathTip: '访问的路由地址，如：user',
        externalPath: '外链地址',
        query: '路由参数',
        iframeQuery: 'iframe 地址',
        isFrame: '是否外链',
        isFrameTip: '选择「是」路由地址为外链；选择「iframe」则内嵌网页',
        visible: '显示状态',
        visibleTip: '隐藏的菜单仍可访问，只是不在侧边栏显示',
        isCache: '是否缓存',
        cache: '缓存',
        isCacheTip: '选择缓存则 keep-alive，刷新页面仍保留状态',
        noCache: '不缓存',
        cascadeDelete: '级联删除',
        cascadeDeleteContent: '确认级联删除选中菜单及其所有子菜单吗？',
        createTime: '创建时间',
        buttonPermissionList: '按钮权限列表',
        placeholder: {
          localIconPlaceholder: '请选择本地图标',
          iconifyIconPlaceholder: '请输入 Iconify 图标名称',
          queryKey: 'Key 不能为空',
          queryValue: 'Value 不能为空',
          queryIframe: '请输入 iframe 地址'
        },
        form: {
          parentId: { required: '请选择上级菜单', invalid: '请选择上级菜单' },
          menuIds: { required: '请选择要删除的菜单', invalid: '请选择要删除的菜单' },
          menuName: { required: '请输入菜单名称', invalid: '菜单名称格式不正确' },
          orderNum: { required: '请输入显示顺序', invalid: '显示顺序格式不正确' },
          perms: { required: '请输入权限标识', invalid: '权限标识格式不正确' },
          path: { required: '请输入路由地址', invalid: '路由地址格式不正确' },
          component: { required: '请输入组件路径', invalid: '组件路径格式不正确' }
        }
      },
      dept: {
        title: '部门',
        parentId: '父部门',
        deptName: '部门名称',
        deptCategory: '部门类别',
        leader: '负责人',
        phone: '联系电话',
        email: '邮箱',
        sort: '排序',
        status: '状态',
        createTime: '创建时间',
        orderNum: '显示顺序',
        expandAll: '展开全部',
        collapseAll: '折叠全部',
        empty: '暂无部门数据',
        addDept: '新增部门',
        editDept: '编辑部门',
        placeholder: {
          defaultLeaderPlaceHolder: '请选择负责人',
          addDataLeaderPlaceHolder: '新增部门时不可选择负责人',
          deptUserIsEmptyLeaderPlaceHolder: '该部门暂无用户，无法选择负责人'
        },
        error: {
          getDeptDataFail: '获取部门数据失败',
          getDeptUserDataFail: '获取部门用户数据失败'
        },
        form: {
          deptId: { required: '请输入部门ID', invalid: '部门ID不能为空' },
          parentId: { required: '请选择父部门', invalid: '请选择父部门' },
          orderNum: { required: '请输入显示顺序', invalid: '显示顺序不能为空' },
          deptName: { required: '请输入部门名称', invalid: '部门名称不能为空' },
          deptCategory: { required: '请输入部门类别', invalid: '部门类别格式不正确' },
          status: { required: '请选择状态', invalid: '请选择状态' },
          phone: { required: '请输入联系电话', invalid: '手机号码格式不正确' },
          email: { required: '请输入邮箱', invalid: '邮箱格式不正确' }
        }
      },
      post: {
        title: '岗位管理',
        listTitle: '岗位信息列表',
        deptTreeTitle: '部门列表',
        emptyDept: '暂无部门信息',
        exportFileName: '岗位信息',
        belongDept: '归属部门',
        postCode: '岗位编码',
        postCategory: '类别编码',
        postName: '岗位名称',
        postSort: '显示顺序',
        status: '状态',
        remark: '备注',
        createTime: '创建时间',
        addPost: '新增岗位信息',
        editPost: '编辑岗位信息',
        form: {
          postId: { required: '请输入岗位ID', invalid: '岗位ID不能为空' },
          deptId: { required: '请选择归属部门', invalid: '归属部门不能为空' },
          postCode: { required: '请输入岗位编码', invalid: '岗位编码不能为空' },
          postCategory: { required: '请输入类别编码', invalid: '类别编码格式不正确' },
          postName: { required: '请输入岗位名称', invalid: '岗位名称不能为空' },
          postSort: { required: '请输入显示顺序', invalid: '显示顺序不能为空' },
          status: { required: '请选择状态', invalid: '状态不能为空' },
          remark: { required: '请输入备注', invalid: '备注格式不正确' }
        }
      },
      dict: {
        title: '字典管理',
        dictName: '字典名称',
        dictTypeTitle: '字典类型',
        dictType: '字典类型',
        dictData: '字典数据',
        addDictType: '新增字典类型',
        addDictData: '新增字典数据',
        exportDictType: '导出字典类型',
        editDictData: '编辑字典数据',
        editDictType: '编辑字典类型',
        refreshDictType: '刷新字典',
        dictTypeIsEmpty: '请先选择字典类型',
        refreshCache: '刷新缓存',
        confirmDeleteDictType: '确认删除字典类型',
        refreshCacheSuccess: '刷新缓存成功',
        remark: '备注',
        data: {
          label: '字典标签',
          value: '字典键值',
          dictSort: '字典排序',
          listClass: '回显样式',
          cssClass: 'CSS 类',
          remark: '备注',
          createTime: '创建时间',
          isDefault: '是否默认'
        },
        form: {
          dictName: { required: '请输入字典名称', invalid: '字典名称不能为空' },
          dictCode: { required: '请输入字典编码', invalid: '字典编码不能为空' },
          dictLabel: { required: '请输入字典标签', invalid: '字典标签不能为空' },
          dictValue: { required: '请输入字典键值', invalid: '字典键值不能为空' },
          dictType: { required: '请输入字典类型', invalid: '字典类型不能为空' },
          listClass: { required: '请选择回显样式', invalid: '请选择回显样式' },
          cssClass: { required: '请输入 CSS 类', invalid: 'CSS 类格式不正确' },
          dictSort: { required: '请输入字典排序', invalid: '字典排序不能为空' },
          remark: { required: '请输入备注', invalid: '备注格式不正确' }
        }
      }
    }
  },
  form: {
    required: '不能为空',
    userName: {
      required: '请输入用户名',
      invalid: '用户名格式不正确'
    },
    phone: {
      required: '请输入手机号',
      invalid: '手机号格式不正确'
    },
    pwd: {
      required: '请输入密码',
      invalid: '密码格式不正确，6-18位字符，包含字母、数字、下划线'
    },
    confirmPwd: {
      required: '请输入确认密码',
      invalid: '两次输入密码不一致'
    },
    code: {
      required: '请输入验证码',
      invalid: '验证码格式不正确'
    },
    email: {
      required: '请输入邮箱',
      invalid: '邮箱格式不正确'
    }
  },
  dropdown: {
    closeCurrent: '关闭',
    closeOther: '关闭其它',
    closeLeft: '关闭左侧',
    closeRight: '关闭右侧',
    closeAll: '关闭所有',
    pin: '固定标签',
    unpin: '取消固定'
  },
  icon: {
    themeConfig: '主题配置',
    themeSchema: '主题模式',
    lang: '切换语言',
    fullscreen: '全屏',
    fullscreenExit: '退出全屏',
    reload: '刷新页面',
    collapse: '折叠菜单',
    expand: '展开菜单',
    pin: '固定',
    unpin: '取消固定'
  },
  datatable: {
    itemCount: '共 {total} 条',
    fixed: {
      left: '左固定',
      right: '右固定',
      unFixed: '取消固定'
    }
  }
};

export default local;
