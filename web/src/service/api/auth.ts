import { request } from '../request';

/** 获取行为验证码（go-captcha），按后端触发策略决定是否返回；username 用于阈值判断 */
export function fetchCaptcha(username?: string) {
  return request<Api.Auth.CaptchaResult>({
    url: '/base/captcha',
    method: 'get',
    params: username ? { username } : {}
  });
}

/**
 * Login（httpOnly cookie 模式：access/refresh 由后端写入 cookie，响应只回 expiresAt）
 *
 * @param data 用户名 + 密码
 */
export function fetchLogin(data: Api.Auth.PwdLoginForm) {
  return request<Api.Auth.LoginToken>({
    url: '/base/login',
    method: 'post',
    data
  });
}

/** Get user info */
export function fetchGetUserInfo() {
  return request<Api.Auth.UserInfo>({ url: '/auth/getUserInfo' });
}

/**
 * Refresh token —— refresh token 由浏览器经 httpOnly cookie 自动携带，无需传参。
 */
export function fetchRefreshToken() {
  return request<Api.Auth.LoginToken>({
    url: '/auth/refreshToken',
    method: 'post',
    data: {}
  });
}

/** Logout - invalidate tokens on server side */
export function fetchLogout() {
  return request({
    url: '/auth/logout',
    method: 'post'
  });
}

/** Register (Soybean 示例保留，后端无端点) */
export function fetchRegister(data: Api.Auth.RegisterForm) {
  return request<Api.Auth.LoginToken>({
    url: '/auth/register',
    method: 'post',
    data
  });
}

/** social login callback (Soybean 示例保留，后端无端点) */
export function fetchSocialLoginCallback(data: Api.Auth.SocialLoginForm) {
  return request({
    url: '/auth/social/callback',
    method: 'post',
    data
  });
}

/** return custom backend error (Soybean 示例保留) */
export function fetchCustomBackendError(code: string, msg: string) {
  return request({ url: '/auth/error', params: { code, msg } });
}
