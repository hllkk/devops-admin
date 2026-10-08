/**
 * 服务器模块共享选项常量
 *
 * 枚举值与后端 model/server 的常量严格对齐；label 为 i18n key，
 * as const 让 label 成为字面量联合类型，便于 $t(o.label) 通过 I18nKey 校验。
 */

/** 资产类型选项 */
export const ASSET_TYPE_OPTIONS = [
  { label: 'page.server.asset.typePhysical', value: 'physical' },
  { label: 'page.server.asset.typeVm', value: 'vm' },
  { label: 'page.server.asset.typeDockerHost', value: 'docker_host' },
  { label: 'page.server.asset.typeDbInstance', value: 'db_instance' },
  { label: 'page.server.asset.typeNetDevice', value: 'net_device' }
] as const;

/** 监控在线状态选项 */
export const MONITOR_STATUS_OPTIONS = [
  { label: 'page.server.asset.monitorOnline', value: 'online' },
  { label: 'page.server.asset.monitorOffline', value: 'offline' },
  { label: 'page.server.asset.monitorUnknown', value: 'unknown' }
] as const;

/** Agent 状态选项(展示用) */
export const AGENT_STATUS_OPTIONS = [
  { label: 'page.server.asset.agentNone', value: 'none' },
  { label: 'page.server.asset.agentInstalling', value: 'installing' },
  { label: 'page.server.asset.agentRunning', value: 'running' },
  { label: 'page.server.asset.agentLost', value: 'lost' }
] as const;

/** 凭据类型选项 */
export const CREDENTIAL_TYPE_OPTIONS = [
  { label: 'page.server.credential.typeSSH', value: 'ssh' },
  { label: 'page.server.credential.typeBMC', value: 'bmc' },
  { label: 'page.server.credential.typeSNMP', value: 'snmp' },
  { label: 'page.server.credential.typeDB', value: 'db' }
] as const;

/** 凭据敏感值掩码(后端 MaskedValue 对齐：回传该值=未修改保留旧明文) */
export const CREDENTIAL_MASKED_VALUE = '******';

/**
 * 凭据表单字段模板(按类型渲染固定键，username 非敏感可回显；密码类为掩码回传)。
 * as const 保留 label 字面量类型(I18nKey 校验)；键集合与后端掩码敏感键对齐。
 */
export const CREDENTIAL_FORM_FIELDS = {
  ssh: [
    { key: 'username', label: 'page.server.credential.fieldUsername', sensitive: false, placeholder: 'root' },
    { key: 'password', label: 'page.server.credential.fieldPassword', sensitive: true, placeholder: '' }
  ],
  bmc: [
    { key: 'username', label: 'page.server.credential.fieldUsername', sensitive: false, placeholder: 'admin' },
    { key: 'password', label: 'page.server.credential.fieldPassword', sensitive: true, placeholder: '' }
  ],
  snmp: [
    { key: 'community', label: 'page.server.credential.fieldCommunity', sensitive: true, placeholder: 'public' },
    { key: 'version', label: 'page.server.credential.fieldVersion', sensitive: false, placeholder: 'v2c' }
  ],
  db: [
    { key: 'username', label: 'page.server.credential.fieldUsername', sensitive: false, placeholder: 'monitor' },
    { key: 'password', label: 'page.server.credential.fieldPassword', sensitive: true, placeholder: '' }
  ]
} as const;

/** 启停状态选项(value 用 1/0，NSelect 不支持 boolean，由搜索组件 computed 转 boolean) */
export const ACTIVE_OPTIONS = [
  { label: 'page.server.common.active', value: 1 },
  { label: 'page.server.common.inactive', value: 0 }
] as const;

/** 环境标签选项(自由字段，给出常用档) */
export const ASSET_ENV_OPTIONS = [
  { label: 'prod', value: 'prod' },
  { label: 'test', value: 'test' },
  { label: 'dev', value: 'dev' }
] as const;

/** 操作系统选项 */
export const ASSET_OS_OPTIONS = [
  { label: 'Linux', value: 'linux' },
  { label: 'Windows', value: 'windows' }
] as const;
