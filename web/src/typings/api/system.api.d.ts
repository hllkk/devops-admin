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
    type DataScope = '1' | '2' | '3' | '4' | '5' | '6';

    /** role */
    type Role = Common.CommonRecord<{
      /** 数据范围（1：全部数据权限 2：自定数据权限 3：本部门数据权限 4：本部门及以下数据权限） */
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
    }>;

    /** user */
    type User = Common.CommonTenantRecord<{
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
      /** 租户ID */
      tenantId: CommonType.IdType;
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
      /** 租户编号 */
      tenantId: CommonType.IdType;
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
  }
}
