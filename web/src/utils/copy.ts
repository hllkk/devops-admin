import { useClipboard } from '@vueuse/core';
import { $t } from '@/locales';

const { copy, isSupported } = useClipboard();

export async function handleCopy(source?: string) {
  if (!isSupported) {
    window.$message?.error($t('common.copyNotSupported'));
    return;
  }

  if (!source) {
    return;
  }

  if (navigator.clipboard && window.isSecureContext) {
    await copy(source);
  } else {
    const range = document.createRange();
    range.selectNode(document.getElementById('tokenDetailInput')!);
    const selection = window.getSelection();
    if (selection?.rangeCount) selection.removeAllRanges();
    selection?.addRange(range);
    document.execCommand('copy');
  }
  window.$message?.success($t('common.copySuccess'));
}
