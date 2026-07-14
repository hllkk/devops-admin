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

    /** password login form（httpOnly cookie 模式：仅用户名密码，captcha 可选兼容） */
    interface PwdLoginForm {
      /** 用户名 */
      username?: string;
      /** 密码 */
      password?: string;
      /** 验证码 */
      code?: string;
      /** 验证码唯一标识 */
      uuid?: string;
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
    }

    interface CaptchaCode {
      /** 是否开启验证码 */
      captchaEnabled: boolean;
      /** 唯一标识 */
      uuid?: string;
      /** 验证码图片 */
      img?: string;
    }
  }
}
