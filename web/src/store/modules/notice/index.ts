import { reactive } from 'vue';
import { defineStore } from 'pinia';
import { SetupStoreId } from '@/enum';
import { fetchGetUnreadNotice, fetchMarkNoticeRead } from '@/service/api/system/notice';
import { stripHtml } from '@/utils/common';

interface NoticeItem {
  noticeId?: number;
  title?: string;
  read: boolean;
  message: any;
  time: string;
}

export const useNoticeStore = defineStore(SetupStoreId.Notice, () => {
  const state: { notices: NoticeItem[] } = reactive({
    notices: []
  });

  const addNotice = (notice: NoticeItem) => {
    // 新消息置顶
    state.notices.unshift(notice);
  };

  const removeNotice = (notice: NoticeItem) => {
    state.notices.splice(state.notices.indexOf(notice), 1);
  };

  // 标记单条已读并同步后端
  const readNotice = async (notice: NoticeItem) => {
    const idx = state.notices.indexOf(notice);
    if (idx < 0) return;
    state.notices[idx].read = true;
    if (notice.noticeId) {
      await fetchMarkNoticeRead([notice.noticeId]);
    }
  };

  // 全部已读并同步后端(空数组=当前用户全部已读)
  const readAll = async () => {
    state.notices.forEach(item => {
      item.read = true;
    });
    await fetchMarkNoticeRead([]);
  };

  const clearNotice = () => {
    state.notices = [];
  };

  // 拉取当前用户未读通知(登录后/铃铛加载时调用),离线消息靠此补齐
  const fetchUnread = async () => {
    const { data, error } = await fetchGetUnreadNotice({ pageNum: 1, pageSize: 50, onlyUnread: true });
    if (error || !data) return;
    const rows = data.rows as (Api.System.Notice & { readAt?: string })[];
    state.notices = rows.map(r => ({
      noticeId: r.noticeId !== undefined ? Number(r.noticeId) : undefined,
      title: r.noticeTitle,
      message: stripHtml(r.noticeContent),
      read: !!r.readAt,
      time: r.createTime ?? ''
    }));
  };

  return {
    state,
    addNotice,
    removeNotice,
    readNotice,
    readAll,
    clearNotice,
    fetchUnread
  };
});
