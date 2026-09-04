import process from 'node:process';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

/**
 * 前端版本号：Docker 构建经 APP_VERSION 注入（compose build.args，与 server ldflags 同源），
 * 开发态回退 package.json 的 version
 */
export function getAppVersion() {
  return process.env.APP_VERSION ?? getPkgVersion();
}

function getPkgVersion() {
  const pkgPath = fileURLToPath(new URL('../../package.json', import.meta.url));
  return (JSON.parse(readFileSync(pkgPath, 'utf-8')) as { version: string }).version;
}
