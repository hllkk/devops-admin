/**
 * Namespace Api
 *
 * All backend api type
 */
declare namespace Api {
  /**
   * namespace System
   *
   * backend api module: "system"
   */
  namespace System {
    /** data scope */
    type DataScope = '1' | '2' | '3' | '4' | '5';

    /** role */
    type Role = Common.CommonRecord<{
      /** 数据范围（1：全部 2：本部门及以下 3：本部门 4：仅本人 5：自定义） */
      dataScope: DataScope;
      /** 部门树选择项是否关联显示 */
      deptCheckStrictly: boolean;
      /** 用户是否存在此角色标识 默认不存在 */
      flag: boolean;
      /** 菜单树选择项是否关联显示 */
      menuCheckStrictly: boolean;
      /** 备注 */
      remark?: string;
      /** 角色ID */
      roleId: CommonType.IdType;
      /** 角色权限字符串 */
      roleKey: string;
      /** 角色名称 */
      roleName: string;
      /** 显示顺序 */
      roleSort: number;
      /** 角色状态（0正常 1停用） */
      status: Common.EnableStatus;
      /** 是否管理员 */
      superAdmin: boolean;
      /** 默认路由(角色登录后默认打开的路由名;主角色决定登录入口) */
      defaultRouter: string;
    }>;

    /** role search params */
    type RoleSearchParams = CommonType.RecordNullable<
      Pick<Api.System.Role, 'roleName' | 'roleKey' | 'status'> & Api.Common.CommonSearchParams
    >;

    /** role operate params */
    type RoleOperateParams = CommonType.RecordNullable<
      Pick<
        Api.System.Role,
        'roleId' | 'roleName' | 'roleKey' | 'roleSort' | 'menuCheckStrictly' | 'deptCheckStrictly'
        | 'dataScope' | 'status' | 'remark' | 'defaultRouter'
      > & { menuIds: CommonType.IdType[]; deptIds: CommonType.IdType[] }
    >;

    /** role list */
    type RoleList = Common.PaginatingQueryRecord<Role>;

    /** role menu tree select */
    type RoleMenuTreeSelect = Common.CommonRecord<{
      checkedKeys: CommonType.IdType[];
      menus: MenuList;
    }>;

    /** role dept tree select */
    type RoleDeptTreeSelect = Common.CommonRecord<{
      checkedKeys: CommonType.IdType[];
      depts: Dept[];
    }>;

    /** user */
    type User = Common.CommonRecord<{
      /** 用户ID */
      userId: CommonType.IdType;
      /** 部门ID */
      deptId: CommonType.IdType;
      /** 部门名称 */
      deptName: string;
      /** 是否超管(任一角色 SuperAdmin;列表超管保护用) */
      superAdmin: boolean;
      /** 用户账号 */
      userName: string;
      /** 用户昵称 */
      nickName: string;
      /** 用户类型（sys_user系统用户） */
      userType: string;
      /** 用户邮箱 */
      email: string;
      /** 手机号码 */
      phonenumber: string;
      /** 用户性别（0男 1女 2未知） */
      sex: string;
      /** 头像地址 */
      avatar: string;
      /** 密码 */
      password: string;
      /** 帐号状态（0正常 1停用） */
      status: Common.EnableStatus;
      /** 最后登录IP */
      loginIp: string;
      /** 最后登录时间 */
      loginDate: Date;
      /** 备注 */
      remark?: string;
    }>;

    /** user search params */
    type UserSearchParams = CommonType.RecordNullable<
      Pick<User, 'deptId' | 'userName' | 'nickName' | 'phonenumber' | 'status'> & {
        roleId: CommonType.IdType;
      } & Common.CommonSearchParams
    >;

    /** user operate params */
    type UserOperateParams = CommonType.RecordNullable<
      Pick<
        User,
        | 'userId'
        | 'deptId'
        | 'userName'
        | 'nickName'
        | 'email'
        | 'phonenumber'
        | 'sex'
        | 'password'
        | 'status'
        | 'remark'
      > & { roleIds: CommonType.IdType[]; postIds: CommonType.IdType[] }
    >;

    /** user profile operate params */
    type UserProfileOperateParams = CommonType.RecordNullable<Pick<User, 'nickName' | 'email' | 'phonenumber' | 'sex'>>;

    /** user password operate params */
    type UserPasswordOperateParams = CommonType.RecordNullable<{
      oldPassword: string;
      newPassword: string;
    }>;

    /** user info */
    type UserInfo = {
      /** user post ids */
      postIds: string[];
      /** user role ids */
      roleIds: string[];
      /** roles */
      roles: Role[];
    };

    /** user list */
    type UserList = Common.PaginatingQueryRecord<User>;

    /** auth role */
    type AuthRole = {
      user: User;
      roles: Role[];
    };

    /** social */
    type Social = Common.CommonRecord<{
      /** 主键 */
      id: CommonType.IdType;
      /** 用户ID */
      userId: CommonType.IdType;
      /** 认证的唯一ID */
      authId: string;
      /** 用户来源 */
      source: string;
      /** 用户的授权令牌 */
      accessToken: string;
      /** 用户的授权令牌的有效期，部分平台可能没有 */
      expireIn: number;
      /** 刷新令牌，部分平台可能没有 */
      refreshToken: string;
      /** 用户的 open id */
      openId: string;
      /** 授权的第三方账号 */
      userName: string;
      /** 授权的第三方昵称 */
      nickName: string;
      /** 授权的第三方邮箱 */
      email: string;
      /** 授权的第三方头像地址 */
      avatar: string;
      /** 平台的授权信息，部分平台可能没有 */
      accessCode: string;
      /** 用户的 unionid */
      unionId: string;
      /** 授予的权限，部分平台可能没有 */
      scope: string;
      /** 个别平台的授权信息，部分平台可能没有 */
      tokenType: string;
      /** id token，部分平台可能没有 */
      idToken: string;
      /** 小米平台用户的附带属性，部分平台可能没有 */
      macAlgorithm: string;
      /** 小米平台用户的附带属性，部分平台可能没有 */
      macKey: string;
      /** 用户的授权code，部分平台可能没有 */
      code: string;
      /** Twitter平台用户的附带属性，部分平台可能没有 */
      oauthToken: string;
      /** Twitter平台用户的附带属性，部分平台可能没有 */
      oauthTokenSecret: string;
    }>;

    /**
     * icon type
     *
     * - "1": iconify icon
     * - "2": local icon
     */
    type IconType = '1' | '2';

    /**
     * menu layout
     *
     * - "0": "默认布局"
     * - "1": "空白布局"
     */
    type MenuLayout = '0' | '1';

    /**
     * menu type
     *
     * - "M": "目录"
     * - "C": "菜单"
     * - "F": "按钮"
     */
    type MenuType = 'M' | 'C' | 'F';

    /**
     * 是否外链
     *
     * - "0": "是"
     * - "1": "否"
     * - "2": "iframe"
     */
    type IsMenuFrame = '0' | '1' | '2';

    type Menu = Common.CommonRecord<{
      /** 菜单 ID */
      menuId: CommonType.IdType;
      /** 父菜单 ID */
      parentId: CommonType.IdType;
      /** 菜单名称 */
      menuName: string;
      /** 显示顺序 */
      orderNum: number;
      /** 路由地址 */
      path: string;
      /** 组件路径 */
      component: string;
      /** 路由参数 */
      queryParam: string;
      /** 是否为外链（0是 1否 2iframe） */
      isFrame: IsMenuFrame;
      /** 是否缓存（0缓存 1不缓存） */
      isCache: Common.EnableStatus;
      /** 菜单类型（M目录 C菜单 F按钮） */
      menuType: MenuType;
      /** 显示状态（0显示 1隐藏） */
      visible: Common.VisibleStatus;
      /** 菜单状态（0正常 1停用） */
      status: Common.EnableStatus;
      /** 权限标识 */
      perms: string;
      /** 菜单图标 */
      icon: string;
      /** 备注 */
      remark?: string;
      /** 业务模块归属(admin/disk/server/gateway;空=未归属,当全局路由) */
      module?: 'admin' | 'disk' | 'server' | 'gateway';
      /** 父菜单名称 */
      parentName: string;
      /** 子菜单 */
      children: MenuList;
      id?: CommonType.IdType;
      label?: string;
    }>;

    /** menu list */
    type MenuList = Menu[];

    /** menu search params */
    type MenuSearchParams = CommonType.RecordNullable<Pick<Menu, 'menuName' | 'status' | 'menuType' | 'parentId'>>;

    /** menu operate params */
    type MenuOperateParams = CommonType.RecordNullable<
      Pick<
        Menu,
        | 'menuId'
        | 'menuName'
        | 'parentId'
        | 'orderNum'
        | 'path'
        | 'component'
        | 'queryParam'
        | 'isFrame'
        | 'isCache'
        | 'menuType'
        | 'visible'
        | 'status'
        | 'perms'
        | 'icon'
        | 'remark'
        | 'module'
      >
    >;

    /** 字典类型 */
    type DictType = Common.CommonRecord<{
      /** 字典主键 */
      dictId: CommonType.IdType;
      /** 字典名称 */
      dictName: string;
      /** 字典类型 */
      dictType: string;
      /** 备注 */
      remark: string;
    }>;

    /** dict type search params */
    type DictTypeSearchParams = CommonType.RecordNullable<
      Pick<Api.System.DictType, 'dictName' | 'dictType'> & Api.Common.CommonSearchParams
    >;

    /** dict type operate params */
    type DictTypeOperateParams = CommonType.RecordNullable<
      Pick<Api.System.DictType, 'dictId' | 'dictName' | 'dictType' | 'remark'>
    >;

    /** dict type list */
    type DictTypeList = Api.Common.PaginatingQueryRecord<DictType>;

    /** 字典数据 */
    type DictData = Common.CommonRecord<{
      /** 样式属性（其他样式扩展） */
      cssClass: string;
      /** 字典编码 */
      dictCode: CommonType.IdType;
      /** 字典标签 */
      dictLabel: string;
      /** 字典排序 */
      dictSort: number;
      /** 字典类型 */
      dictType: string;
      /** 字典键值 */
      dictValue: string;
      /** 是否默认（Y是 N否） */
      isDefault: Common.YesOrNoStatus;
      /** 表格回显样式 */
      listClass: NaiveUI.ThemeColor;
      /** 备注 */
      remark: string;
      /** 是否多语言 */
      isI18n?: boolean;
      /** 多语言标识 */
      i18nKey: App.I18n.I18nKey;
    }>;

    /** dict data search params */
    type DictDataSearchParams = CommonType.RecordNullable<
      Pick<Api.System.DictData, 'dictLabel' | 'dictType'> & Api.Common.CommonSearchParams
    >;

    /** dict data operate params */
    type DictDataOperateParams = CommonType.RecordNullable<
      Pick<
        Api.System.DictData,
        | 'dictCode'
        | 'dictSort'
        | 'dictLabel'
        | 'dictValue'
        | 'dictType'
        | 'cssClass'
        | 'listClass'
        | 'isDefault'
        | 'remark'
      >
    >;

    /** dict data list */
    type DictDataList = Api.Common.PaginatingQueryRecord<DictData>;

    /** dept */
    type Dept = Api.Common.CommonRecord<{
      /** 部门id */
      deptId: CommonType.IdType;
      /** 父部门id */
      parentId: CommonType.IdType;
      /** 祖级列表 */
      ancestors: string;
      /** 部门名称 */
      deptName: string;
      /** 部门类别编码 */
      deptCategory: string;
      /** 显示顺序 */
      orderNum: number;
      /** 负责人 */
      leader: number;
      /** 联系电话 */
      phone: string;
      /** 邮箱 */
      email: string;
      /** 部门状态（0正常 1停用） */
      status: Common.EnableStatus;
      /** 子部门 */
      children: Dept[];
    }>;

    /** dept search params */
    type DeptSearchParams = CommonType.RecordNullable<
      Pick<Api.System.Dept, 'deptName' | 'status'> & Api.Common.CommonSearchParams
    >;

    /** dept operate params */
    type DeptOperateParams = CommonType.RecordNullable<
      Pick<
        Api.System.Dept,
        'deptId' | 'parentId' | 'deptName' | 'deptCategory' | 'orderNum' | 'leader' | 'phone' | 'email' | 'status'
      >
    >;

    /** dept list */
    type DeptList = Api.Common.PaginatingQueryRecord<Dept>;

    /** post */
    type Post = Common.CommonRecord<{
      /** 岗位ID */
      postId: CommonType.IdType;
      /** 部门id */
      deptId: CommonType.IdType;
      /** 岗位编码 */
      postCode: string;
      /** 类别编码 */
      postCategory: string;
      /** 岗位名称 */
      postName: string;
      /** 显示顺序 */
      postSort: number;
      /** 状态（0正常 1停用） */
      status: Common.EnableStatus;
      /** 备注 */
      remark: string;
    }>;

    /** post search params */
    type PostSearchParams = CommonType.RecordNullable<
      Pick<Api.System.Post, 'deptId' | 'postCode' | 'postName' | 'status'> & {
        belongDeptId: CommonType.IdType;
      } & Api.Common.CommonSearchParams
    >;

    /** post operate params */
    type PostOperateParams = CommonType.RecordNullable<
      Pick<
        Api.System.Post,
        'postId' | 'deptId' | 'postCode' | 'postCategory' | 'postName' | 'postSort' | 'status' | 'remark'
      >
    >;

    /** post list */
    type PostList = Api.Common.PaginatingQueryRecord<Post>;

    /** 通知公告类型 */
    type NoticeType = '1' | '2';

    /** notice */
    type Notice = Common.CommonRecord<{
      /** 公告ID */
      noticeId: CommonType.IdType;
      /** 公告标题 */
      noticeTitle: string;
      /** 公告类型 */
      noticeType: System.NoticeType;
      /** 公告内容 */
      noticeContent: string;
      /** 公告状态 */
      status: Common.EnableStatus;
      /** 创建者 */
      createByName: string;
      /** 备注 */
      remark: string;
    }>;

    /** notice search params */
    type NoticeSearchParams = CommonType.RecordNullable<
      Pick<Api.System.Notice, 'noticeTitle' | 'noticeType'> & Api.Common.CommonSearchParams
    >;

    /** notice operate params */
    type NoticeOperateParams = CommonType.RecordNullable<
      Pick<Api.System.Notice, 'noticeId' | 'noticeTitle' | 'noticeType' | 'noticeContent' | 'status'>
    >;

    /** notice list */
    type NoticeList = Api.Common.PaginatingQueryRecord<Notice>;

    /** 设备类型 */
    type DeviceType = 'pc' | 'android' | 'ios' | 'xcx';

    /** social source */
    type SocialSource =
      | 'maxkey'
      | 'topiam'
      | 'qq'
      | 'weibo'
      | 'gitee'
      | 'baidu'
      | 'csdn'
      | 'coding'
      | 'oschina'
      | 'alipay_wallet'
      | 'wechat_open'
      | 'wechat_mp'
      | 'wechat_enterprise'
      | 'gitlab'
      | 'wecom'
      | 'github';

    /** oss */
    type Oss = Common.CommonRecord<{
      /** 对象存储主键 */
      ossId: CommonType.IdType;
      /** 文件名 */
      fileName: string;
      /** 原名 */
      originalName: string;
      /** 文件后缀名 */
      fileSuffix: string;
      /** URL地址 */
      url: string;
      /** 扩展属性 */
      ext1: string;
      /** 服务商 */
      service: string;
      /** 创建者名称 */
      createByName: string;
    }>;

    /** LDAP 配置：对齐后端 SysLdapConfig */
    type LdapSettingConfig = {
      /** 是否启用 LDAP */
      enabled: boolean;
      /** 服务器地址 */
      host: string;
      /** 端口 */
      port: number;
      /** 是否 LDAPS */
      useSSL: boolean;
      /** 管理员绑定 DN */
      bindDN: string;
      /** 管理员绑定密码 */
      bindPass: string;
      /** 搜索 Base DN */
      baseDN: string;
      /** 用户过滤器，%s 替换为登录用户名 */
      filter: string;
      /** 用户名属性 */
      attrUsername: string;
      /** 昵称属性 */
      attrNickname: string;
      /** 邮箱属性 */
      attrEmail: string;
      /** 自动创建本地用户 */
      autoCreate: boolean;
    };

    /** 通知配置：邮件通知 + Webhook，对齐后端 SysNotifyConfig */
    type NotifySettingConfig = {
      /** 启用邮件通知 */
      emailEnabled: boolean;
      /** SMTP 服务器地址 */
      emailHost: string;
      /** SMTP 端口 */
      emailPort: number;
      /** SMTP 认证用户名 */
      emailUsername: string;
      /** SMTP 认证密码 */
      emailPassword: string;
      /** 发件人邮箱地址 */
      emailFromAddr: string;
      /** 发件人显示名称 */
      emailFromName: string;
      /** 加密方式：none / ssl / starttls */
      emailSSLMode: string;
      /** 启用 Webhook 通知 */
      webhookEnabled: boolean;
      /** Webhook 推送地址 */
      webhookUrl: string;
      /** Webhook 签名密钥（可选） */
      webhookSecret: string;
    };

    /** 网盘配置：对齐后端 SysDiskConfig */
    type DiskSettingConfig = {
      /** 最大上传大小（数值） */
      maxUploadSize: number;
      /** 上传大小单位：MB / GB / TB */
      maxUploadSizeUnit: string;
      /** 默认存储配额（数值） */
      storageQuota: number;
      /** 配额单位：MB / GB / TB */
      storageQuotaUnit: string;
      /** 允许上传扩展名（逗号分隔，空=允许全部） */
      allowedExtensions: string;
      /** 禁止上传扩展名（逗号分隔，优先级高于允许） */
      blockedExtensions: string;
      /** 回收站自动清理天数 */
      recycleBinRetentionDays: number;
      /** 网盘名称 */
      diskName: string;
      /** Logo URL */
      diskLogo: string;
      /** 启用 OnlyOffice */
      onlyOfficeEnabled: boolean;
      /** Document Server 地址 */
      onlyOfficeServerUrl: string;
      /** JWT 签名密钥 */
      onlyOfficeTokenSecret: string;
      /** 回调地址（OnlyOffice 容器可访问的后端地址） */
      onlyOfficeCallbackUrl: string;
    };

    /** 认证配置：第三方登录 OAuth2 + 账号功能，对齐后端 SysAuthConfig */
    type AuthSettingConfig = {
      /** 是否开放注册 */
      registerEnabled: boolean;
      /** 是否开放找回密码 */
      resetPwdEnabled: boolean;
      /** 企业微信 */
      wecomEnabled: boolean;
      wecomCorpId: string;
      wecomAgentId: number;
      wecomClientId: string;
      wecomClientSecret: string;
      wecomCallbackUrl: string;
      wecomDomainFileName: string;
      wecomDomainFileContent: string;
      /** 微信开放平台 */
      wechatEnabled: boolean;
      wechatClientId: string;
      wechatClientSecret: string;
      wechatCallbackUrl: string;
      /** Gitee */
      giteeEnabled: boolean;
      giteeClientId: string;
      giteeClientSecret: string;
      giteeCallbackUrl: string;
      /** GitHub */
      githubEnabled: boolean;
      githubClientId: string;
      githubClientSecret: string;
      githubCallbackUrl: string;
    };

    /** 系统设置：聚合配置（GET/PUT /system/setting 的请求与响应体） */
    type Setting = {
      general?: GeneralSettingConfig;
      security?: SecuritySettingConfig;
      ldap?: LdapSettingConfig;
      disk?: DiskSettingConfig;
      notify?: NotifySettingConfig;
      auth?: AuthSettingConfig;
    };

    /** 公开系统设置（登录页使用，免鉴权脱敏：系统信息 + 验证码开关） */
    type PublicSetting = {
      // 系统信息(sys_general_config)
      systemName: string;
      systemDescription: string;
      logoUrl: string;
      faviconUrl: string;
      // 验证码(sys_security_config.Captcha*,登录页验证码渲染用)
      captchaEnabled: boolean;
      captchaType: string;
      captchaOpen: number;
      keyLong: number;
      imgWidth: number;
      imgHeight: number;
      // 认证公开字段(仅开关，不含密钥)
      registerEnabled: boolean;
      resetPwdEnabled: boolean;
      wecomEnabled: boolean;
      wechatEnabled: boolean;
      giteeEnabled: boolean;
      githubEnabled: boolean;
    };

    /** 通用配置(系统信息 + 日志清理) */
    type GeneralSettingConfig = {
      systemName: string;
      systemDescription: string;
      logoUrl: string;
      faviconUrl: string;
      /** 登录日志保留天数 */
      loginLogRetentionDays: number;
      /** 操作日志保留天数 */
      operationLogRetentionDays: number;
      /** 导入/重置用户的默认密码 */
      defaultPassword: string;
    };

    /** 安全配置:对齐后端 SysSecurityConfig 六段字段 */
    type SecuritySettingConfig = {
      // 验证码(Captcha*):登录链路验证码生成用
      captchaEnabled: boolean;
      captchaType: string;
      captchaOpen: number;
      captchaTimeout: number;
      captchaTolerance: number;
      keyLong: number;
      imgWidth: number;
      imgHeight: number;
      // 密码复杂度(Password*)
      passwordMinLength: number;
      passwordRequireUppercase: boolean;
      passwordRequireLowercase: boolean;
      passwordRequireDigit: boolean;
      passwordRequireSpecial: boolean;
      // 登录失败锁定(LoginFailLock*)
      loginFailLockCount: number;
      loginFailLockTime: number;
      // 访问控制 - IP 校验(IpValidation*)
      ipValidationEnabled: boolean;
      ipValidationMode: string;
      ipBlacklist: string;
      ipWhitelist: string;
      // 限流(Limit*)
      limitEnable: boolean;
      limitWindow: number;
      limitCount: number;
      // 密码过期(PwdExpire*)
      pwdExpireEnable: boolean;
      pwdExpireDays: number;
    };

    /** 定时任务执行器类型 */
    type TimedTaskExecutorType = 'method' | 'http';

    /** 定时任务 */
    type SysTimedTask = Common.CommonRecord<{
      id: CommonType.IdType;
      name: string;
      description: string;
      spec: string;
      withSeconds: boolean;
      executorType: TimedTaskExecutorType;
      methodName: string;
      params: Record<string, unknown> | null;
      httpUrl: string;
      httpMethod: string;
      httpHeader: Record<string, unknown> | null;
      httpBody: string;
      httpAllowPrivate: boolean;
      enabled: boolean;
      nextRunAt: string | null;
    }>;

    /** 定时任务创建/更新参数 */
    type SysTimedTaskOperateParams = CommonType.RecordNullable<
      Pick<
        SysTimedTask,
        'id' | 'name' | 'description' | 'spec' | 'withSeconds' | 'executorType' | 'methodName' | 'params' | 'httpUrl' | 'httpMethod' | 'httpHeader' | 'httpBody' | 'httpAllowPrivate' | 'enabled'
      >
    >;

    /** 定时任务搜索参数 */
    type SysTimedTaskSearchParams = CommonType.RecordNullable<
      Pick<SysTimedTask, 'name' | 'executorType' | 'enabled'> & Api.Common.CommonSearchParams
    >;

    /** 定时任务列表 */
    type SysTimedTaskList = Api.Common.PaginatingQueryRecord<SysTimedTask>;

    /** 定时任务执行日志触发类型 */
    type TimedTaskLogTriggerType = 'auto' | 'manual';

    /** 定时任务执行日志状态 */
    type TimedTaskLogStatus = 'success' | 'fail' | 'timeout';

    /** 定时任务执行日志 */
    type SysTimedTaskLog = Common.CommonRecord<{
      id: CommonType.IdType;
      taskId: CommonType.IdType;
      taskName: string;
      triggerType: TimedTaskLogTriggerType;
      startedAt: string;
      finishedAt: string;
      durationMs: number;
      status: TimedTaskLogStatus;
      errorMsg: string;
      output: string;
    }>;

    /** 定时任务执行日志搜索参数 */
    type SysTimedTaskLogSearchParams = CommonType.RecordNullable<
      Pick<SysTimedTaskLog, 'taskId' | 'status'> & Api.Common.CommonSearchParams
    >;

    /** 定时任务执行日志列表 */
    type SysTimedTaskLogList = Api.Common.PaginatingQueryRecord<SysTimedTaskLog>;

    /** 已注册方法 */
    type RegisteredMethod = {
      name: string;
      description: string;
    };

    /** 已注册方法列表 */
    type RegisteredMethodList = {
      methods: RegisteredMethod[];
    };
  }
}
