/**
 * 网关接入地址展示重写。
 *
 * 后端下发的 litellm public-url 在 dev 环境常是 127.0.0.1/localhost 占位，
 * 用户从别的机器访问时该地址不可达；按当前页面访问的 host 重写后即可直连。
 * 生产(PRD)的 public-url 由 .env 注入正式域名/内网 IP，不属于本地占位，原样下发不受影响。
 */

/** 视为本地占位的 host(dev 配置) */
const LOCAL_HOSTS = new Set(['localhost', '127.0.0.1', '0.0.0.0', '[::1]', '::1']);

/** dev 本地占位地址按当前访问 host 重写(端口/路径保留)；非占位地址原样返回 */
export function rewriteGatewayUrl(url: string): string {
  if (!url) return url;
  try {
    const parsed = new URL(url);
    if (!LOCAL_HOSTS.has(parsed.hostname)) return url;
    const host = window.location.hostname;
    if (host && !LOCAL_HOSTS.has(host)) parsed.hostname = host;
    // URL.toString() 会给无路径地址补尾斜杠，展示/复制时去掉
    return parsed.toString().replace(/\/$/, '');
  } catch {
    return url;
  }
}

type McpConnectConfig = NonNullable<Api.Gateway.MCPConnectConfig['config']>;

/** 客户端接入配置 JSON 内的网关地址同步重写(只动 mcpServers[*].url，不误伤其它字段) */
export function rewriteGatewayConfig(config: McpConnectConfig): McpConnectConfig {
  const servers: McpConnectConfig['mcpServers'] = {};
  for (const [name, server] of Object.entries(config.mcpServers)) {
    servers[name] = server?.url ? { ...server, url: rewriteGatewayUrl(server.url) } : server;
  }
  return { ...config, mcpServers: servers };
}
