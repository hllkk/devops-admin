export function getLocalIcons() {
  const svgIcons = import.meta.glob('/src/assets/svg-icon/*.svg');

  const keys = Object.keys(svgIcons)
    .map(item => item.split('/').at(-1)?.replace('.svg', '') || '')
    .filter(Boolean);

  return keys;
}

export function getLocalMenuIcons() {
  const svgIcons = import.meta.glob('/src/assets/svg-icon/menu/*.svg');

  const keys = Object.keys(svgIcons)
    .map(item => item.split('/').at(-1)?.replace('.svg', '') || '')
    .filter(Boolean);

  return keys;
}

/** AI 能力图标(MCP/Skill 等业务卡片用)：lucide 精选集，localIcon 名带 ai- 目录前缀 */
export function getLocalAiIcons() {
  const svgIcons = import.meta.glob('/src/assets/svg-icon/ai/*.svg');

  const keys = Object.keys(svgIcons)
    .map(item => item.split('/').at(-1)?.replace('.svg', '') || '')
    .filter(Boolean)
    .map(name => `ai-${name}`);

  return keys;
}
