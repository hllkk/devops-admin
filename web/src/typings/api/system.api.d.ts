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
    /** role */
    type Role = Common.CommonRecord<{
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
    }>;

    /** role search params */
    type RoleSearchParams = CommonType.RecordNullable<
      Pick<Api.System.Role, 'roleName' | 'roleKey' | 'status'> & Api.Common.CommonSearchParams
    >;

    /** role operate params */
    type RoleOperateParams = CommonType.RecordNullable<
      Pick<
        Api.System.Role,
        'roleId' | 'roleName' | 'roleKey' | 'roleSort' | 'menuCheckStrictly' | 'status' | 'remark'
      > & { menuIds: CommonType.IdType[] }
    >;

    /** role list */
    type RoleList = Common.PaginatingQueryRecord<Role>;

    /** role menu tree select */
    type RoleMenuTreeSelect = Common.CommonRecord<{
      checkedKeys: CommonType.IdType[];
      menus: MenuList;
    }>;

    /** user */
    type User = Common.CommonRecord<{
      /** 用户ID */
      userId: CommonType.IdType;
      /** 部门ID */
      deptId: CommonType.IdType;
      /** 部门名称 */
      deptName: string;
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
      | 'dingtalk'
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

    /** 系统设置：聚合配置（GET/PUT /system/setting 的请求与响应体） */
    type Setting = {
      general?: GeneralSettingConfig;
      security?: SecuritySettingConfig;
      // 阶段二扩展：authentication / ldap / notify / disk
    };

    /** 公开系统设置（登录页使用，脱敏） */
    type PublicSetting = {
      systemName: string;
      systemDescription: string;
      logoUrl: string;
      faviconUrl: string;
      enableVerifyCode: boolean;
      verifyCodeType?: string;
      verifyCodeLen?: number;
      verifyCodeExp?: number;
      verifyCodeTokenExp?: number;
      verifyInaccuracy?: number;
      enableWecom?: boolean;
      enableWechat?: boolean;
      enableGitee?: boolean;
      enableGithub?: boolean;
    };

    /** 通用配置 */
    type GeneralSettingConfig = {
      systemName: string;
      systemDescription: string;
      logoUrl: string;
      faviconUrl: string;
      userDefaultPassword: string;
      userDefaultRole: string | null;
      enableVerifyCode: boolean;
      verifyCodeType: string;
      verifyCodeLen: number;
      verifyCodeExp: number;
      verifyCodeTokenExp: number;
      verifyInaccuracy: number;
      loginLogRetentionDays: number;
      operationLogRetentionDays: number;
    };

    /** 安全配置 */
    type SecuritySettingConfig = {
      passwordMinLength: number;
      passwordRequireUppercase: boolean;
      passwordRequireLowercase: boolean;
      passwordRequireDigit: boolean;
      passwordRequireSpecial: boolean;
      loginFailLockCount: number;
      loginFailLockTime: number;
      ipValidationEnabled: boolean;
      ipValidationMode: string;
      ipBlacklist: string;
      ipWhitelist: string;
    };
  }
}
