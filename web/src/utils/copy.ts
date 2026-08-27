import { useClipboard } from '@vueuse/core';
import { $t } from '@/locales';

// legacy: true —— 非安全上下文(http/IP 直连访问)下无 navigator.clipboard 时，
// 由 vueuse 自动降级为 execCommand(内部临时 textarea)，不依赖页面内任何元素
const { copy, isSupported } = useClipboard({ legacy: true });

export async function handleCopy(source?: string) {
  if (!source) {
    return;
  }

  if (!isSupported.value) {
    window.$message?.error($t('common.copyNotSupported'));
    return;
  }

  try {
    await copy(source);
    window.$message?.success($t('common.copySuccess'));
  } catch {
    window.$message?.error($t('common.copyNotSupported'));
  }
}
