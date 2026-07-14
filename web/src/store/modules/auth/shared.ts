import { localStg } from '@/utils/storage';

/** 登录态信号：httpOnly cookie 模式下 token 不进 JS，用 isAuthenticated 布尔判定是否已登录。 */
export function getToken(): string {
  const isAuthenticated = localStg.get('isAuthenticated');
  return isAuthenticated ? 'authenticated' : '';
}

/** 清除登录态本地信号（真正 token 由后端清 httpOnly cookie）。 */
export function clearAuthStorage() {
  localStg.remove('isAuthenticated');
  localStg.remove('tokenExpiresAt');
}
