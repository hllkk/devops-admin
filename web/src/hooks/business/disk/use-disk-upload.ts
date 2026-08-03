import { computed, ref } from 'vue';
import { fetchCancelUpload, fetchCheckUpload, fetchEnsureFolders, fetchMergeUpload, fetchUploadChunk } from '@/service/api/disk';
import { chunkMd5, computeSampleHashes, getChunkSize } from '@/utils/disk-hash';

/**
 * 网盘上传引擎(模块级单例)。
 * toolbar 触发上传、transfer-panel 展示进度,共用同一份 tasks。
 * 流程:采样哈希(quickHash+strongHash) → check(秒传/续传) → 并发上传缺失分片 → merge → 完成。
 * 支持:文件夹上传(透传 relativePath)、暂停/继续、重试、取消,与 service/disk/disk_upload.go 契约对齐。
 */

const CONCURRENCY = 3;

const tasks = ref<Api.Disk.UploadTask[]>([]);
const panelVisible = ref(false);
let autoHideTimer: ReturnType<typeof setTimeout> | null = null;

const ACTIVE_STATUS: Api.Disk.UploadTaskStatus[] = ['pending', 'hashing', 'uploading', 'merging'];
const uploadingCount = computed(() => tasks.value.filter(t => ACTIVE_STATUS.includes(t.status)).length);

/** 每任务控制器:暂停/继续/取消/重试所需的可变状态 + 原始入参(重试用) */
interface Ctrl {
  paused: boolean;
  cancelled: boolean;
  pausePromise: Promise<void> | null;
  resolvePause: (() => void) | null;
  identifier?: string; // quickHash,取消后端会话用
  file?: File;
  currentDirectory: string;
  relativePath?: string;
  onFileDone?: () => void;
  /** 进入 uploading 态的时间戳(ms),算平均速度用;重试时重置 */
  startedAt?: number;
  /** 累计暂停时长(ms),从活动时间里扣除 */
  pausedMs: number;
  /** 当前这次暂停的开始时间戳(ms),恢复时累加进 pausedMs */
  pausedAt?: number;
}
const controllers = new Map<string, Ctrl>();

// 仅作 instanceof 哨兵:区分"用户取消"与真实异常。带一个标记成员,避免被 no-extraneous-class 判为空类。
class CancelledSentinel {
  readonly sentinel = true;
}

/** 由「活动时长 = now - startedAt - pausedMs」算平均速度(bytes/s)与剩余秒数 */
function calcSpeedEta(startedAt: number | undefined, pausedMs: number, uploaded: number, total: number) {
  if (!startedAt) return { speed: 0, remainingTime: 0 };
  const sec = Math.max((Date.now() - startedAt - pausedMs) / 1000, 0);
  const speed = sec > 0 ? uploaded / sec : 0;
  const remainingTime = speed > 0 ? Math.max(0, (total - uploaded) / speed) : 0;
  return { speed: Math.round(speed), remainingTime: Math.round(remainingTime) };
}

function genId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}
function updateTask(id: string, patch: Partial<Api.Disk.UploadTask>) {
  const t = tasks.value.find(x => x.id === id);
  if (t) Object.assign(t, patch);
}
/** 若处于暂停态,worker 在此挂起,直到 resume 唤醒 */
function ctrlWaitPause(c: Ctrl): Promise<void> {
  if (!c.paused) return Promise.resolve();
  if (!c.pausePromise) c.pausePromise = new Promise<void>(r => { c.resolvePause = r; });
  return c.pausePromise;
}
function ctrlResume(c: Ctrl) {
  c.paused = false;
  if (c.resolvePause) {
    c.resolvePause();
    c.resolvePause = null;
    c.pausePromise = null;
  }
}
function showPanel() {
  panelVisible.value = true;
  if (autoHideTimer) {
    clearTimeout(autoHideTimer);
    autoHideTimer = null;
  }
}
function hidePanel() {
  panelVisible.value = false;
}
function togglePanel() {
  if (panelVisible.value) hidePanel();
  else showPanel();
}
/** 全部完成 3s 后自动收起面板 */
function maybeAutoHide() {
  if (uploadingCount.value === 0) {
    if (autoHideTimer) clearTimeout(autoHideTimer);
    autoHideTimer = setTimeout(() => {
      hidePanel();
      autoHideTimer = null;
    }, 3000);
  }
}

export function useDiskUpload() {
  /** 单文件上传主体(可复用同 id+ctrl 重试) */
  async function runUpload(id: string, ctrl: Ctrl) {
    const { file, currentDirectory, relativePath, onFileDone } = ctrl;
    if (!file) return;
    updateTask(id, { status: 'hashing', progress: 0, errorMsg: undefined, transferredSize: 0 });
    try {
      const { quickHash, strongHash, midHash } = await computeSampleHashes(file);
      if (ctrl.cancelled) throw new CancelledSentinel();
      ctrl.identifier = quickHash;
      const chunkSize = getChunkSize(file.size);
      const totalChunks = Math.max(1, Math.ceil(file.size / chunkSize));

      const { data: check, error: checkErr } = await fetchCheckUpload({
        identifier: quickHash,
        quickHash,
        strongHash,
        midHash,
        fileName: file.name,
        totalSize: file.size,
        totalChunks,
        chunkSize,
        currentDirectory,
        relativePath
      });
      if (ctrl.cancelled) throw new CancelledSentinel();
      if (checkErr || !check) {
        updateTask(id, { status: 'error', errorMsg: '检测失败' });
        maybeAutoHide();
        return;
      }
      if (check.pass) {
        // 秒传(H1):不传分片,直接调 merge(instant=true) 在目标目录建引用节点 + 源 ref_count++
        updateTask(id, { status: 'merging', progress: 100 });
        const { data: instantMerge, error: instantErr } = await fetchMergeUpload({
          identifier: quickHash,
          fileName: file.name,
          totalSize: file.size,
          totalChunks,
          currentDirectory,
          relativePath,
          quickHash,
          strongHash,
          midHash,
          instant: true
        });
        if (instantErr || !instantMerge) {
          updateTask(id, { status: 'error', errorMsg: '秒传落库失败' });
          maybeAutoHide();
          return;
        }
        updateTask(id, { status: 'success', progress: 100, transferredSize: file.size, remainingTime: 0 });
        onFileDone?.();
        maybeAutoHide();
        return;
      }

      const uploadId = check.uploadId;
      const done = new Set(check.resume || []);
      const pending: number[] = [];
      for (let i = 0; i < totalChunks; i += 1) if (!done.has(i)) pending.push(i);
      updateTask(id, { status: 'uploading' });
      ctrl.startedAt = Date.now(); // 进入上传态,开始计时(重试时重置)
      ctrl.pausedMs = 0;
      let uploaded = (totalChunks - pending.length) * chunkSize; // 已续传字节(近似)
      // 续传:立即把进度推到断点,避免进度条从 0 显示造成"从头传"假象
      updateTask(id, {
        progress: file.size > 0 ? Math.min(99, Math.floor((uploaded / file.size) * 100)) : 0,
        transferredSize: uploaded,
        speed: 0,
        remainingTime: 0
      });

      // 字节级进度:进行中分片的实时已传字节(并发累加用),配合 reportProgress 节流上报
      const chunkLoaded = new Map<number, number>();
      let lastReport = 0;
      const REPORT_INTERVAL = 100; // ms,节流避免高频刷 Vue 卡 UI
      const reportProgress = (force = false) => {
        const now = Date.now();
        if (!force && now - lastReport < REPORT_INTERVAL) return;
        lastReport = now;
        let live = uploaded;
        for (const v of chunkLoaded.values()) live += v;
        const { speed, remainingTime } = calcSpeedEta(ctrl.startedAt, ctrl.pausedMs, live, file.size);
        updateTask(id, {
          progress: file.size > 0 ? Math.min(99, Math.floor((live / file.size) * 100)) : 0,
          transferredSize: live,
          speed,
          remainingTime
        });
      };

      let cursor = 0;
      const worker = async () => {
        for (;;) {
          if (ctrl.cancelled) throw new CancelledSentinel();
          await ctrlWaitPause(ctrl);
          if (ctrl.cancelled) throw new CancelledSentinel();
          const idx = cursor;
          cursor += 1;
          if (idx >= pending.length) return;
          const i = pending[idx];
          const start = i * chunkSize;
          const end = Math.min(start + chunkSize, file.size);
          const blob = file.slice(start, end);
          const hash = await chunkMd5(blob);
          const { error } = await fetchUploadChunk(
            { uploadId, chunkNumber: i, chunkHash: hash, file: blob },
            e => {
              // 该分片实时已传字节(限幅到分片大小,防超界)
              chunkLoaded.set(i, Math.min(e.loaded || 0, end - start));
              reportProgress();
            }
          );
          if (error) {
            chunkLoaded.delete(i);
            throw new Error(`分片 ${i} 上传失败`);
          }
          chunkLoaded.delete(i);
          uploaded += end - start;
          reportProgress(true); // 分片完成:强制精确上报一次
        }
      };
      const workers = Array.from({ length: Math.min(CONCURRENCY, pending.length) }, () => worker());
      await Promise.all(workers);

      updateTask(id, { status: 'merging', progress: 100 });
      const { data: merge, error: mergeErr } = await fetchMergeUpload({
        identifier: quickHash,
        fileName: file.name,
        totalSize: file.size,
        totalChunks,
        currentDirectory,
        relativePath,
        quickHash,
        strongHash,
        midHash
      });
      if (ctrl.cancelled) throw new CancelledSentinel();
      if (mergeErr || !merge) {
        updateTask(id, { status: 'error', errorMsg: '合并失败' });
        maybeAutoHide();
        return;
      }
      updateTask(id, { status: 'success', progress: 100, transferredSize: file.size });
      onFileDone?.();
      maybeAutoHide();
      controllers.delete(id); // 成功:释放控制器
    } catch (e) {
      if (e instanceof CancelledSentinel) return; // 取消:任务由 cancelTask 移除,不标错
      updateTask(id, { status: 'error', errorMsg: (e as Error).message || '上传失败' });
      maybeAutoHide();
      // 出错时保留控制器,供 retryTask 重试
    }
  }

  function uploadOne(file: File, currentDirectory: string, relativePath: string | undefined, onFileDone?: () => void) {
    const id = genId();
    const ctrl: Ctrl = {
      paused: false,
      cancelled: false,
      pausePromise: null,
      resolvePause: null,
      file,
      currentDirectory,
      relativePath,
      onFileDone,
      pausedMs: 0
    };
    controllers.set(id, ctrl);
    tasks.value.push({ id, name: file.name, size: file.size, progress: 0, status: 'hashing', relativePath });
    showPanel();
    // eslint-disable-next-line @typescript-eslint/no-floating-promises
    runUpload(id, ctrl);
  }

  /** 批量上传入口。
   *  文件夹上传时:先调 ensure-folders 预建所有目录(含空目录),再逐文件上传;
   *  每个 File 的 webkitRelativePath 透传给后端 merge 阶段懒建中间目录(双保险)。
   *  dirs 为空(单文件上传或降级)时跳过预建。 */
  async function uploadFiles(files: File[], currentDirectory: string, dirs?: string[], onFileDone?: () => void) {
    // 文件夹预建目录(含空目录):best effort,失败不阻断后续文件上传(merge 阶段仍会懒建中间目录)
    if (dirs && dirs.length) {
      try {
        await fetchEnsureFolders({ currentDirectory, paths: dirs });
      } catch {
        /* 预建失败:merge 阶段仍懒建中间目录,仅空目录可能丢失 */
      }
    }
    for (const f of files) {
      const rel = (f as File & { webkitRelativePath?: string }).webkitRelativePath || undefined;
      uploadOne(f, currentDirectory, rel, onFileDone);
    }
  }

  function pauseTask(id: string) {
    const ctrl = controllers.get(id);
    if (!ctrl) return;
    ctrl.paused = true;
    ctrl.pausedAt = Date.now(); // 记录暂停起点,恢复时累加进 pausedMs,从活动时间里扣除
    updateTask(id, { status: 'paused' });
  }
  function resumeTask(id: string) {
    const ctrl = controllers.get(id);
    if (!ctrl) return;
    if (ctrl.pausedAt) {
      ctrl.pausedMs += Date.now() - ctrl.pausedAt;
      ctrl.pausedAt = undefined;
    }
    ctrlResume(ctrl);
    updateTask(id, { status: 'uploading' });
  }
  function retryTask(id: string) {
    const ctrl = controllers.get(id);
    if (!ctrl) return;
    ctrl.cancelled = false;
    ctrl.paused = false;
    // eslint-disable-next-line @typescript-eslint/no-floating-promises
    runUpload(id, ctrl);
  }
  async function cancelTask(id: string) {
    const ctrl = controllers.get(id);
    if (ctrl) {
      ctrl.cancelled = true;
      ctrlResume(ctrl); // 唤醒可能 paused 的 worker,使其退出
      if (ctrl.identifier) {
        try {
          await fetchCancelUpload(ctrl.identifier);
        } catch {
          /* 最佳努力取消,忽略 */
        }
      }
      controllers.delete(id);
    }
    tasks.value = tasks.value.filter(x => x.id !== id);
  }
  function pauseAll() {
    tasks.value.forEach(t => {
      if (ACTIVE_STATUS.includes(t.status)) pauseTask(t.id);
    });
  }
  function resumeAll() {
    tasks.value.forEach(t => {
      if (t.status === 'paused') resumeTask(t.id);
    });
  }

  /** 清除已完成/失败的任务 */
  function clearFinished() {
    tasks.value = tasks.value.filter(t => t.status !== 'success' && t.status !== 'error');
  }
  function removeTask(id: string) {
    tasks.value = tasks.value.filter(t => t.id !== id);
  }

  return {
    tasks,
    uploadingCount,
    panelVisible,
    uploadFiles,
    pauseTask,
    resumeTask,
    retryTask,
    cancelTask,
    pauseAll,
    resumeAll,
    clearFinished,
    removeTask,
    showPanel,
    hidePanel,
    togglePanel
  };
}
