declare namespace Api {
  /**
   * namespace Auth
   *
   * backend api module: "auth"
   */
  namespace Auth {
    /** base login form（保留 clientId/grantType/captcha 供 social-callback 编译兼容） */
    interface LoginForm {
      /** 客户端 ID */
      clientId?: string;
      /** 授权类型 */
      grantType?: string;
      /** 验证码 */
      code?: string;
      /** 唯一标识 */
      uuid?: string;
    }

    /** password login form（httpOnly cookie 模式：用户名密码 + go-captcha 验证码） */
    interface PwdLoginForm {
      /** 用户名 */
      username?: string;
      /** 密码 */
      password?: string;
      /** go-captcha 验证码会话 ID */
      captchaId?: string;
      /** go-captcha 用户答案（JSON 字符串：click 为点坐标数组、slide 为 {x,y}、rotate 为 {angle}） */
      captcha?: string;
    }

    /** register form (Soybean 示例保留，后端无端点) */
    interface RegisterForm extends LoginForm {
      /** 用户名 */
      username?: string;
      /** 密码 */
      password?: string;
      /** 确认密码 */
      confirmPassword?: string;
      /** 用户类型 */
      userType?: string;
    }

    /** social login form */
    interface SocialLoginForm extends LoginForm {
      /** 授权码 */
      socialCode?: string;
      /** 授权状态 */
      socialState?: string;
      /** 来源 */
      source?: string;
    }

    /** 登录/刷新响应：token 仅存 httpOnly cookie 不回传，expiresAt 为 access token 过期毫秒时间戳 */
    interface LoginToken {
      expiresAt: number;
    }

    /** userinfo */
    interface UserInfo {
      /** 用户信息 */
      user?: Api.System.User & {
        /** 所属角色 */
        roles: Api.System.Role[];
      };
      /** 角色列表 */
      roles: string[];
      /** 菜单权限 */
      permissions: string[];
      /** 可见应用(业务模块标识 admin/disk/server/gateway;后端按模块菜单权限聚合随 getUserInfo 下发) */
      apps: string[];
      /** 默认路由(主角色 DefaultRouter;登录入口) */
      defaultRouter: string;
    }

    /** go-captcha 行为验证码生成结果 */
    interface CaptchaResult {
      /** 当前是否要求验证码（false 时其余字段为空） */
      captchaEnabled: boolean;
      /** 验证码类型：image 传统字符图片 | click | slide | rotate 行为验证 */
      type?: string;
      /** 验证码会话 ID */
      captchaId?: string;
      /** 主图 base64 */
      masterImage?: string;
      /** 拼图块 base64（slide） */
      tileImage?: string;
      /** 提示缩略图 base64（click/rotate） */
      thumbImage?: string;
      /** slide 拼图块初始 X */
      thumbX?: number;
      /** slide 拼图块初始 Y */
      thumbY?: number;
      /** slide 拼图块宽度 */
      thumbWidth?: number;
      /** slide 拼图块高度 */
      thumbHeight?: number;
      /** rotate 缩略图初始角度 */
      angle?: number;
      /** rotate 缩略图尺寸 */
      thumbSize?: number;
    }
  }
}
