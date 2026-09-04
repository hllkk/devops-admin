import type { Plugin } from 'vite';
import { getAppVersion } from '../config';

export function setupHtmlPlugin(buildTime: string) {
  const plugin: Plugin = {
    name: 'html-plugin',
    apply: 'build',
    transformIndexHtml(html) {
      // appVersion meta：前端产物自证版本（排查缓存新旧用，与 server ldflags 同源）
      return html.replace(
        '<head>',
        `<head>\n    <meta name="buildTime" content="${buildTime}">\n    <meta name="appVersion" content="${getAppVersion()}">`
      );
    }
  };

  return plugin;
}
