import { watch } from 'vue';
import { useEventSource } from '@vueuse/core';
import { useNoticeStore } from '@/store/modules/notice';
import { stripHtml } from '@/utils/common';

/**
 * 初始化 SSE(鉴权走 httpOnly cookie x-token,EventSource 同源/withCredentials 自动携带)
 *
 * @param url - SSE 地址(/resource/sse)
 */
export const initSSE = (url: string) => {
  if (import.meta.env.VITE_APP_SSE === 'N') {
    return;
  }
  const { data, error } = useEventSource(url, [], {
    withCredentials: true,
    autoReconnect: {
      retries: 5,
      delay: 5000,
      onFailed() {
        // eslint-disable-next-line no-console
        console.warn('Failed to connect to SSE after 5 attempts.');
      }
    }
  });

  watch(error, () => {
    if (!error.value || error.value?.isTrusted) {
      return;
    }
    // eslint-disable-next-line no-console
    console.error('SSE connection error:\n', error.value);
    error.value = null;
  });

  watch(data, () => {
    if (!data.value) return;

    // 后端推送 JSON,统一走 message 通道,用 payload.type 区分:
    // 通知: { noticeId, noticeTitle, noticeContent, noticeType }
    // 告警: { type:'timedTask:alert', taskId, name, error, time }
    let payload: {
      type?: string;
      noticeId?: number;
      noticeTitle?: string;
      noticeContent?: string;
      taskId?: number;
      name?: string;
      error?: string;
    } = {};
    try {
      payload = JSON.parse(data.value);
    } catch {
      payload = { noticeTitle: '消息', noticeContent: data.value };
    }
    const isAlert = payload.type === 'timedTask:alert' || payload.taskId !== undefined;
    const title = payload.noticeTitle || (isAlert ? '定时任务告警' : '消息');
    // 公告内容是富文本 HTML，预览/弹窗统一转纯文本，避免铃铛里看到 "<p>...</p>" 字面量
    const content = payload.noticeContent
      ? stripHtml(payload.noticeContent)
      : isAlert
        ? `${payload.name ?? ''}:${payload.error ?? ''}`
        : '';
    useNoticeStore().addNotice({
      noticeId: payload.noticeId,
      title,
      message: content,
      read: false,
      time: new Date().toLocaleString()
    });
    window.$notification?.create({
      title,
      content,
      type: isAlert ? 'error' : 'success',
      duration: 3000
    });
    data.value = null;
  });
};
