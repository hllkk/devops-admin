import { ref, onMounted, onUnmounted, watch, nextTick, type Ref } from 'vue';

/**
 * 无限滚动 hook — 监听滚动容器的 scroll 事件
 * 当滚动到底部附近（距底部 ≤ threshold px）时触发 loadMore
 *
 * 优势：不受容器嵌套层级影响，只需知道哪个 HTMLElement 是滚动容器
 *
 * 多驱动触发（修复大视口/4K 下"首屏内容不足以产生滚动条 → scroll 事件永不触发 → 懒加载卡死"）：
 * - scroll 事件：用户主动滚动接近底部
 * - 首屏加载完成检查：第一页 loading 结束后主动检查（覆盖刷新/切目录/搜索/排序等首屏填满）
 * - 加载完成后重检查：一页仍填不满视口则继续加载，直到可滚动或 !hasMore
 * - ResizeObserver：窗口/视口尺寸变化时重检查（如窗口从标准拖到 4K）
 *
 * 竞态保护：isLoading 期间跳过 check，避免在第一页数据未就绪时误触分页，
 * 与 getFileList 的缓存重置/realTotal 设置竞争导致 hasMore 错算。
 */
interface UseInfiniteScrollOptions {
  /** 加载更多数据的回调 */
  onLoadMore: () => Promise<void>;
  /** 是否还有更多数据可加载 */
  hasMore: Ref<boolean>;
  /** 滚动容器 ref — 内部有 overflow-y-auto 的 div 或 NDataTable wrapper */
  scrollContainerRef: Ref<HTMLElement | null>;
  /** 触发加载的距离阈值（px），距底部多少 px 时触发，默认 200 */
  threshold?: number;
  /**
   * 首屏/第一页是否正在加载（消除竞态）。
   * loading 期间 check 跳过；loading 由 true→false 时主动触发一次首屏检查。
   */
  isLoading?: Ref<boolean>;
}

export function useInfiniteScroll(options: UseInfiniteScrollOptions) {
  const { onLoadMore, hasMore, scrollContainerRef, threshold = 200, isLoading } = options;

  const loadingMore = ref(false);

  /**
   * 主动检查是否需要加载更多，供多个触发点（scroll/首屏/加载后/resize）复用。
   * 受 hasMore、loadingMore、isLoading 多重保护，不会无限触发：
   * 每次加载后内容增高，一旦 distanceToBottom > threshold（填满视口）即自然停止。
   */
  function check() {
    const el = scrollContainerRef.value;
    if (!el || !hasMore.value || loadingMore.value) return;
    // 首屏加载中跳过：避免与 getFileList 的缓存重置/realTotal 设置竞争
    if (isLoading?.value) return;

    // 判断是否接近底部：scrollTop + clientHeight + threshold >= scrollHeight
    const distanceToBottom = el.scrollHeight - el.scrollTop - el.clientHeight;
    if (distanceToBottom <= threshold) {
      triggerLoadMore();
    }
  }

  async function triggerLoadMore() {
    if (loadingMore.value || !hasMore.value) return;
    loadingMore.value = true;
    try {
      await onLoadMore();
    } finally {
      loadingMore.value = false;
      // 加载完成后重检查：若仍填不满视口则继续加载，
      // 形成"自动连续加载直到可滚动或 !hasMore"，覆盖大视口/4K 首屏。
      nextTick(check);
    }
  }

  let boundElement: HTMLElement | null = null;
  let resizeObserver: ResizeObserver | undefined;

  function bindScrollListener() {
    // 先解绑旧元素
    if (boundElement) {
      boundElement.removeEventListener('scroll', check);
      boundElement = null;
    }

    const el = scrollContainerRef.value;
    if (!el) return;

    el.addEventListener('scroll', check, { passive: true });
    boundElement = el;

    // 绑定后立即检查（首屏数据已就绪时生效；isLoading 期间会被 check 内部跳过）
    nextTick(check);
  }

  onMounted(() => {
    nextTick(bindScrollListener);
    // 观察 documentElement（非容器本身）：网格容器用 max-height，内容不足时其实际高度
    // 不随窗口变化；而 documentElement 必然随 window 尺寸变化（含纯高度拖大），
    // 此时 clientHeight 增大 → distanceToBottom 减小 → 正确触发补加载。
    resizeObserver = new ResizeObserver(() => nextTick(check));
    resizeObserver.observe(document.documentElement);
  });

  // 首屏/第一页加载完成（loading: true→false）后主动检查：
  // 覆盖刷新、SPA 切换目录、搜索/筛选/排序等场景，确保大视口首屏填满。
  // 这是安全的检查时机——此时第一页数据已就绪，realTotal/hasMore 已正确设置。
  if (isLoading) {
    watch(isLoading, loading => {
      if (!loading) nextTick(check);
    });
  }

  // 滚动容器可能延迟渲染或切换（grid ↔ list），需要动态重新绑定
  watch(scrollContainerRef, () => {
    nextTick(bindScrollListener);
  });

  onUnmounted(() => {
    if (boundElement) {
      boundElement.removeEventListener('scroll', check);
    }
    resizeObserver?.disconnect();
  });

  return { loadingMore };
}
