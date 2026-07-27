export function isPC() {
  const agents = ['Android', 'iPhone', 'webOS', 'BlackBerry', 'SymbianOS', 'Windows Phone', 'iPad', 'iPod'];

  const isMobile = agents.some(agent => window.navigator.userAgent.includes(agent));

  return !isMobile;
}

/** 是否运行在企业微信客户端 WebView 内(UA 含 wxwork),用于免登分支判定 */
export function isWecomWebview() {
  return window.navigator.userAgent.toLowerCase().includes('wxwork');
}
