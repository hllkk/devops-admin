import { $t } from '@/locales';

/**
 * 复制文本到剪贴板，失败抛错(调用方据此给用户真实反馈，不谎报"已复制")。
 *
 * 1. 安全上下文(https/localhost)优先 Clipboard API，失败(权限被拒等)降级 execCommand；
 * 2. http/IP 直连等非安全上下文无 navigator.clipboard，走 execCommand 路线：
 *    - 选区载体优先用调用方给的可见元素(与 SoyDisk 分享复制同构，copy 事件必触发)；
 *      没有可见元素时用离屏 textarea——clipboard.js 方案，元素保持渲染仅移出视口(left:-9999px)，
 *      严禁 opacity/display 隐藏：部分内核对不可见元素的复制会假成功；
 *    - 通过拦截 copy 事件用 clipboardData.setData 显式写入数据，不依赖浏览器读取选区内容
 *      (API Key 行显示的是掩码，也要能复制出原文)；
 *    - 成功判据 = execCommand 返回 true 且 copy 事件真实触发，杜绝"已复制"假成功。
 */
export async function copyTextToClipboard(text: string, selectEl?: HTMLElement | null): Promise<void> {
  if (!text) return;
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      // 降级 execCommand
    }
  }

  if (!document.hasFocus()) {
    // 文档失焦时 execCommand('copy') 必然失败，提前抛错
    throw new Error('document is not focused');
  }

  const selection = window.getSelection();
  let textarea: HTMLTextAreaElement | null = null;

  if (selectEl) {
    // 可见元素作选区载体(仅触发 copy 流程，实际写入值由下方 setData 决定)
    const range = document.createRange();
    range.selectNodeContents(selectEl);
    selection?.removeAllRanges();
    selection?.addRange(range);
  } else {
    textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.setAttribute('readonly', '');
    textarea.style.cssText = [
      'position:fixed',
      'top:0',
      'left:-9999px',
      'width:2em',
      'height:2em',
      'padding:0',
      'border:none',
      'outline:none',
      'box-shadow:none',
      'background:transparent',
      'pointer-events:none'
    ].join(';');
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    textarea.setSelectionRange(0, text.length);
  }

  // 拦截 copy 事件显式写入数据：数据进不进剪贴板只取决于事件是否触发，与选区内容无关
  let eventFired = false;
  const onCopy = (e: ClipboardEvent) => {
    eventFired = true;
    e.clipboardData?.setData('text/plain', text);
    e.preventDefault();
  };
  document.addEventListener('copy', onCopy, true);

  let copied = false;
  try {
    copied = document.execCommand('copy');
  } finally {
    document.removeEventListener('copy', onCopy, true);
    textarea?.remove();
    selection?.removeAllRanges();
  }

  if (!copied || !eventFired) {
    throw new Error(`execCommand copy failed (ok=${copied}, eventFired=${eventFired})`);
  }
}

/**
 * 复制被浏览器拒绝时的兜底：选中目标元素文本，引导用户 Ctrl+C 手动复制。
 * 手动选中复制不依赖任何剪贴板 API，非安全上下文/权限受限场景都可用。
 */
export function selectText(el?: HTMLElement | null) {
  if (!el) return;
  const range = document.createRange();
  range.selectNodeContents(el);
  const selection = window.getSelection();
  selection?.removeAllRanges();
  selection?.addRange(range);
}

/** 带 toast 的复制(表格/列表等通用场景)，返回是否复制成功 */
export async function handleCopy(source?: string) {
  if (!source) return false;
  try {
    await copyTextToClipboard(source);
    window.$message?.success($t('common.copySuccess'));
    return true;
  } catch {
    window.$message?.error($t('common.copyFailed'));
    return false;
  }
}
