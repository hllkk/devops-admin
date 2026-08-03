import { computed, ref } from 'vue';
import { fetchCancelUpload, fetchCheckUpload, fetchEnsureFolders, fetchMergeUpload, fetchUploadChunk } from '@/service/api/disk';
import { fetchGetSetting } from '@/service/api/system/setting';
import { chunkMd5, computeSampleHashes, getChunkSize } from '@/utils/disk-hash';

/**
 * 网盘上传引擎(模块级单例)。
 * toolbar 触发上传、transfer-panel 展示进度,共用同一份 tasks。
 * 流程:采样哈希(quickHash+strongHash) → check(秒传/续传) → 并发上传缺失分片 → merge → 完成。
 * 支持:文件夹上传(透传 relativePath)、暂停/继续、重试、取消,与 service/disk/disk_upload.go 契约对齐。
 */

/** 单文件分片并发数(0=按设备 navigator.hardwareConcurrency 自适应,下限2上限4);由 sys_disk_config.maxChunkConcurrency 配置 */
let chunkConcurrency = 0;
/** 单分片上传失败重试次数(0=不重试,默认3);由 sys_disk_config.maxChunkRetries 配置 */
let chunkMaxRetries = 3;

/** 当前生效的分片并发数:配置>0 用配置,否则按 CPU 核数自适应 */
function getConcurrency(): number {
  return chunkConcurrency > 0 ? chunkConcurrency : Math.min(4, Math.max(2, (navigator.hardwareConcurrency || 4) - 2));
}

/** 文件级并发池:多文件上传时限制同时跑的任务数(由 sys_disk_config.maxConcurrentUploads 配置,0=不限)。
 *  旧实现 uploadFiles 循环 fire-and-forget,所有文件同时启动 → N×分片并发压垮浏览器/后端。
 *  现入队受控调度,超出的排队等待(pending 态可见)。retry 绕过池直接跑(单任务手动重试,瞬时+1可接受)。 */
let maxConcurrent = 0;
let concurrentConfigLoaded = false;
interface QueuedTask {
  id: string;
  ctrl: Ctrl;
}
const uploadQueue: QueuedTask[] = [];
let activeUploads = 0;

const tasks = ref<Api.Disk.UploadTask[]>([]);
const panelVisible = ref(false);
let autoHideTimer: ReturnType<typeof setTimeout> | null = null;

/** 活跃状态(在跑):不含 pending(排队中)。pending 不算活跃,不触发自动收起 */
const ACTIVE_STATUS: Api.Disk.UploadTaskStatus[] = ['hashing', 'uploading', 'merging'];
/** 未完成任务数(含排队 pending + 活跃 + 暂停),用于 badge 与自动收起判断 */
const uploadingCount = computed(
  () => tasks.value.filter(t => t.status !== 'success' && t.status !== 'error').length
);

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
  /** 进入 uploading 态的时间戳(ms),算速度用;重试时重置 */
  startedAt?: number;
  /** 累计暂停时长(ms),从活动时间里扣除 */
  pausedMs: number;
  /** 当前这次暂停的开始时间戳(ms),恢复时累加进 pausedMs */
  pausedAt?: number;
  /** EMA 平滑后的速度(bytes/s),供 UI 展示 */
  emaSpeed?: number;
  /** 上次速度采样时间戳(ms);恢复后置 undefined 触发重采,避免暂停时长污染 */
  speedLastTs?: number;
  /** 上次速度采样已传字节,配合 speedLastTs 算段增量 */
  speedLastUploaded?: number;
  /** 退避 sleep 的定时器句柄,cancel 时可提前 clearTimeout */
  sleepTimer?: ReturnType<typeof setTimeout>;
  /** 退避 sleep 的 reject 句柄,cancel 时调用以立即打断等待 */
  rejectSleep?: (e: CancelledSentinel) => void;
}
const controllers = new Map<string, Ctrl>();

// 仅作 instanceof 哨兵:区分"用户取消"与真实异常。带一个标记成员,避免被 no-extraneous-class 判为空类。
class CancelledSentinel {
  readonly sentinel = true;
}

/** EMA 速度平滑系数与采样间隔下限(对齐 remote:α=0.25,抑制原始速度抖动) */
const SPEED_EMA_ALPHA = 0.25;
const SPEED_MIN_SAMPLE_MS = 100;
/**
 * 由「本采样段增量字节 / 段时长」算瞬时速度,再 EMA 平滑得展示速度与剩余秒数。
 * 旧实现用「总字节/总活动时长」的平均速度,随时间越来越迟钝;EMA 反应更跟手。
 * 采样点(speedLastTs/speedLastUploaded)与 emaSpeed 挂在 Ctrl 上,每任务独立。
 */
function calcSpeedEta(ctrl: Ctrl, live: number, total: number) {
  const { startedAt, emaSpeed, speedLastTs, speedLastUploaded } = ctrl;
  if (!startedAt) return { speed: 0, remainingTime: 0 };
  const now = Date.now();
  // 首采(或恢复后重采):仅记录基线,不产速度,避免单点噪声
  if (speedLastTs === undefined || speedLastUploaded === undefined) {
    ctrl.speedLastTs = now;
    ctrl.speedLastUploaded = live;
    return { speed: Math.round(emaSpeed || 0), remainingTime: 0 };
  }
  const dt = now - speedLastTs;
  // 间隔过短:沿用上次 EMA,等累积足够增量再采样
  if (dt < SPEED_MIN_SAMPLE_MS) {
    return {
      speed: Math.round(emaSpeed || 0),
      remainingTime: emaSpeed ? Math.round((total - live) / emaSpeed) : 0
    };
  }
  const instant = ((live - speedLastUploaded) * 1000) / dt; // 瞬时 bytes/s
  // 停顿(无新字节):EMA 衰减,避免卡在旧峰值;有增量:标准 EMA
  const newEma =
    instant > 0 ? SPEED_EMA_ALPHA * instant + (1 - SPEED_EMA_ALPHA) * (emaSpeed || 0) : (emaSpeed || 0) * 0.5;
  ctrl.emaSpeed = newEma;
  ctrl.speedLastTs = now;
  ctrl.speedLastUploaded = live;
  const remainingTime = newEma > 0 ? Math.max(0, (total - live) / newEma) : 0;
  return { speed: Math.round(newEma), remainingTime: Math.round(remainingTime) };
}

function genId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}
/** 文件夹聚合重算节流定时器(folderId -> timer),避免高频 progress 更新刷爆聚合计算 */
const folderRecomputeTimers = new Map<string, ReturnType<typeof setTimeout>>();
/** 文件夹聚合速度 EMA 状态(folderId -> {ema,lastTs,lastTransferred}) */
const folderSpeedState = new Map<string, { ema: number; lastTs: number; lastTransferred: number }>();

function updateTask(id: string, patch: Partial<Api.Disk.UploadTask>) {
  const t = tasks.value.find(x => x.id === id);
  if (!t) return;
  Object.assign(t, patch);
  // 子文件状态/进度变化:节流触发所属文件夹聚合重算
  if (t.folderId && !t.isFolderAgg) scheduleFolderRecompute(t.folderId);
}

/** 节流重算文件夹聚合(150ms 内多次更新合并为一次),降低大文件夹下的计算频率 */
function scheduleFolderRecompute(folderId: string) {
  if (folderRecomputeTimers.has(folderId)) return;
  const timer = setTimeout(() => {
    folderRecomputeTimers.delete(folderId);
    recomputeFolder(folderId);
  }, 150);
  folderRecomputeTimers.set(folderId, timer);
}

/**
 * 重算文件夹聚合条目:遍历子文件汇总 completedCount/folderTransferredSize/progress/速度/状态。
 * 子文件不单独占顶层列表行(由 UI 按 folderId 折叠),聚合条目是其唯一顶层展示。
 */
function recomputeFolder(folderId: string) {
  const agg = tasks.value.find(t => t.id === folderId && t.isFolderAgg);
  if (!agg) return;
  const children = tasks.value.filter(t => t.folderId === folderId && !t.isFolderAgg);
  if (!children.length) return;
  let completed = 0;
  let failed = 0;
  let active = 0;
  let transferred = 0;
  for (const c of children) {
    if (c.status === 'success') completed += 1;
    else if (c.status === 'error') failed += 1;
    else if (ACTIVE_STATUS.includes(c.status) || c.status === 'paused') active += 1;
    transferred += c.transferredSize || 0;
  }
  const total = agg.folderTotalSize || agg.size || 0;
  const progress = total > 0 ? Math.min(100, Math.floor((transferred / total) * 100)) : 0;
  const speed = computeFolderSpeed(folderId, transferred);
  // 聚合状态:全成功→success;仍有活动→uploading;否则有失败→error;余下 pending
  let status: Api.Disk.UploadTaskStatus;
  if (completed === children.length) status = 'success';
  else if (active > 0) status = 'uploading';
  else if (failed > 0) status = 'error';
  else status = 'pending';
  Object.assign(agg, {
    completedCount: completed,
    folderTransferredSize: transferred,
    progress,
    speed,
    remainingTime: speed > 0 ? Math.round((total - transferred) / speed) : 0,
    status
  });
  if (status === 'success' || status === 'error') maybeAutoHide();
}

/** 文件夹聚合速度 EMA:基于 folderTransferredSize 增量(比累加子文件瞬时速度稳,小文件快子项 speed 常 0) */
function computeFolderSpeed(folderId: string, transferred: number): number {
  let st = folderSpeedState.get(folderId);
  if (!st) {
    st = { ema: 0, lastTs: 0, lastTransferred: 0 };
    folderSpeedState.set(folderId, st);
  }
  const now = Date.now();
  if (!st.lastTs) {
    st.lastTs = now;
    st.lastTransferred = transferred;
    return Math.round(st.ema);
  }
  const dt = now - st.lastTs;
  if (dt < 150) return Math.round(st.ema);
  const instant = ((transferred - st.lastTransferred) * 1000) / dt;
  st.ema = instant > 0 ? 0.4 * instant + 0.6 * st.ema : st.ema * 0.5;
  st.lastTs = now;
  st.lastTransferred = transferred;
  return Math.round(st.ema);
}
/** 清理文件夹聚合瞬时状态(速度 EMA + 重算节流定时器),文件夹移除时调用防 Map 内存泄漏 */
function cleanupFolderState(folderId: string) {
  folderSpeedState.delete(folderId);
  const timer = folderRecomputeTimers.get(folderId);
  if (timer) {
    clearTimeout(timer);
    folderRecomputeTimers.delete(folderId);
  }
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
/** 指数退避 + jitter:base * 2^attempt * (0.8~1.2),约 1s/2s/4s,避免雪崩重试 */
function retryDelay(attempt: number): number {
  const jitter = 0.8 + Math.random() * 0.4;
  return Math.round(1000 * 2 ** attempt * jitter);
}
/** 取消可控的 sleep:退避期间 cancelTask 调 rejectSleep 立即打断,无需轮询 cancelled */
function ctrlSleep(c: Ctrl, ms: number): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    c.sleepTimer = setTimeout(() => {
      c.sleepTimer = undefined;
      c.rejectSleep = undefined;
      resolve();
    }, ms);
    c.rejectSleep = e => {
      clearTimeout(c.sleepTimer);
      c.sleepTimer = undefined;
      c.rejectSleep = undefined;
      reject(e);
    };
  });
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

/** 首次上传时从 sys_disk_config 读上传配置(并发/分片并发/重试);读失败用默认,且只加载一次 */
async function ensureConcurrentConfig() {
  if (concurrentConfigLoaded) return;
  concurrentConfigLoaded = true; // 标记已加载,失败也不重试(避免每次上传都打失败请求)
  try {
    const { data } = await fetchGetSetting();
    if (data?.disk) {
      if (data.disk.maxConcurrentUploads > 0) maxConcurrent = data.disk.maxConcurrentUploads;
      if (data.disk.maxChunkConcurrency > 0) chunkConcurrency = data.disk.maxChunkConcurrency;
      if (data.disk.maxChunkRetries >= 0) chunkMaxRetries = data.disk.maxChunkRetries;
    }
  } catch {
    /* 配置读失败:用默认(不限/自适应/3) */
  }
}

/** 重载配置(配置页保存后调用,热更新):重置加载标志并重读,新任务立即按新配置调度 */
function reloadConfig() {
  concurrentConfigLoaded = false;
  // eslint-disable-next-line @typescript-eslint/no-floating-promises
  ensureConcurrentConfig();
}

/** 有活跃上传任务时,离开/刷新页面弹原生确认(避免误关导致上传中断丢失) */
if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', (e: BeforeUnloadEvent) => {
    if (uploadingCount.value > 0) {
      e.preventDefault();
      e.returnValue = ''; // Chromium 系需赋值才弹确认
    }
  });
}

export function useDiskUpload() {
  /** 单文件上传主体(可复用同 id+ctrl 重试) */
  async function runUpload(id: string, ctrl: Ctrl, onEnd?: () => void) {
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
        updateTask(id, { status: 'error', errorMsg: checkErr?.message || '检测失败' });
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
          updateTask(id, { status: 'error', errorMsg: instantErr?.message || '秒传落库失败' });
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
      ctrl.emaSpeed = 0; // 速度采样基线重置,避免上一轮/重试残留
      ctrl.speedLastTs = undefined;
      ctrl.speedLastUploaded = undefined;
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
        const { speed, remainingTime } = calcSpeedEta(ctrl, live, file.size);
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
          if (ctrl.cancelled) throw new CancelledSentinel();
          // 分片级重试:网络抖动/超时重试 N 次 + 指数退避,替代旧"单次失败即整任务标错"
          let success = false;
          for (let attempt = 0; attempt <= chunkMaxRetries; attempt += 1) {
            if (ctrl.cancelled) throw new CancelledSentinel();
            const { error } = await fetchUploadChunk(
              { uploadId, chunkNumber: i, chunkHash: hash, file: blob },
              e => {
                // 该分片实时已传字节(限幅到分片大小,防超界)
                chunkLoaded.set(i, Math.min(e.loaded || 0, end - start));
                reportProgress();
              }
            );
            if (!error) {
              success = true;
              break;
            }
            if (attempt < chunkMaxRetries) {
              await ctrlSleep(ctrl, retryDelay(attempt)); // 退避;期间取消则 reject CancelledSentinel
            }
          }
          if (!success) {
            chunkLoaded.delete(i);
            throw new Error(`分片 ${i} 上传失败(已重试 ${chunkMaxRetries} 次)`);
          }
          chunkLoaded.delete(i);
          uploaded += end - start;
          reportProgress(true); // 分片完成:强制精确上报一次
        }
      };
      const workers = Array.from({ length: Math.min(getConcurrency(), pending.length) }, () => worker());
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
        updateTask(id, { status: 'error', errorMsg: mergeErr?.message || '合并失败' });
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
    } finally {
      onEnd?.(); // 释放并发池槽(取消/失败/成功均触发),续调度下一个
    }
  }

  /** 建上传任务(占位 pending 排队态)+ 控制器,返回 id/ctrl;不立即 runUpload,交调度池 */
  function createUploadTask(
    file: File,
    currentDirectory: string,
    relativePath: string | undefined,
    onFileDone?: () => void,
    folderId?: string
  ): { id: string; ctrl: Ctrl } {
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
    tasks.value.push({ id, name: file.name, size: file.size, progress: 0, status: 'pending', relativePath, folderId });
    showPanel();
    return { id, ctrl };
  }

  /** 调度队列:并发池有空闲(或不限)时取队首启动 runUpload,任务结束 onEnd 释放槽并续调度 */
  function scheduleNext() {
    while (uploadQueue.length && (maxConcurrent === 0 || activeUploads < maxConcurrent)) {
      const { id, ctrl } = uploadQueue.shift()!;
      activeUploads += 1;
      // eslint-disable-next-line @typescript-eslint/no-floating-promises
      runUpload(id, ctrl, () => {
        activeUploads -= 1;
        scheduleNext();
      });
    }
  }
  /** 入队受控调度(替代旧 fire-and-forget):建 task 后入队,排队态可见,调度池空闲时启动 */
  function enqueueUpload(item: {
    file: File;
    currentDirectory: string;
    relativePath: string | undefined;
    onFileDone?: () => void;
    folderId?: string;
  }) {
    const { id, ctrl } = createUploadTask(item.file, item.currentDirectory, item.relativePath, item.onFileDone, item.folderId);
    uploadQueue.push({ id, ctrl });
    scheduleNext();
  }

  /** 建文件夹聚合条目:占位 task(isFolderAgg)汇总同顶层 relativePath 的子文件 */
  function createFolderAgg(folderName: string, groupFiles: File[], currentDirectory: string, onFileDone?: () => void) {
    const folderId = genId();
    const folderTotalSize = groupFiles.reduce((s, f) => s + f.size, 0);
    tasks.value.push({
      id: folderId,
      name: folderName,
      size: folderTotalSize,
      progress: 0,
      status: 'pending',
      isFolderAgg: true,
      folderId,
      folderName,
      totalCount: groupFiles.length,
      completedCount: 0,
      folderTotalSize,
      folderTransferredSize: 0,
      relativePath: folderName
    });
    showPanel();
    for (const f of groupFiles) {
      const rel = (f as File & { webkitRelativePath?: string }).webkitRelativePath || undefined;
      enqueueUpload({ file: f, currentDirectory, relativePath: rel, onFileDone, folderId });
    }
  }

  /** 批量上传入口。
   *  文件夹上传时:先调 ensure-folders 预建所有目录(含空目录),再逐文件上传;
   *  每个 File 的 webkitRelativePath 透传给后端 merge 阶段懒建中间目录(双保险)。
   *  dirs 为空(单文件上传或降级)时跳过预建。 */
  async function uploadFiles(files: File[], currentDirectory: string, dirs?: string[], onFileDone?: () => void) {
    await ensureConcurrentConfig(); // 先加载并发配置,再入队(避免首次上传在配置未就绪时全量启动)
    // 文件夹预建目录(含空目录):best effort,失败不阻断后续文件上传(merge 阶段仍会懒建中间目录)
    if (dirs && dirs.length) {
      try {
        await fetchEnsureFolders({ currentDirectory, paths: dirs });
      } catch {
        /* 预建失败:merge 阶段仍懒建中间目录,仅空目录可能丢失 */
      }
    }
    // 按 webkitRelativePath 顶层段分组:同顶层的归一个文件夹聚合条目,避免大文件夹刷屏
    const folderGroups = new Map<string, File[]>();
    const standalone: File[] = [];
    for (const f of files) {
      const rel = (f as File & { webkitRelativePath?: string }).webkitRelativePath || '';
      const segs = rel.split('/');
      if (rel && segs.length > 1) {
        const top = segs[0]!;
        const arr = folderGroups.get(top);
        if (arr) arr.push(f);
        else folderGroups.set(top, [f]);
      } else {
        standalone.push(f);
      }
    }
    for (const f of standalone) {
      const rel = (f as File & { webkitRelativePath?: string }).webkitRelativePath || undefined;
      enqueueUpload({ file: f, currentDirectory, relativePath: rel, onFileDone });
    }
    for (const [folderName, groupFiles] of folderGroups) {
      createFolderAgg(folderName, groupFiles, currentDirectory, onFileDone);
    }
  }

  function pauseTask(id: string) {
    const agg = tasks.value.find(t => t.id === id);
    if (agg?.isFolderAgg) {
      // 聚合:暂停所有活动子文件
      tasks.value.forEach(c => {
        if (c.folderId === id && !c.isFolderAgg && ACTIVE_STATUS.includes(c.status)) pauseTask(c.id);
      });
      return;
    }
    const ctrl = controllers.get(id);
    if (!ctrl) return;
    ctrl.paused = true;
    ctrl.pausedAt = Date.now(); // 记录暂停起点,恢复时累加进 pausedMs,从活动时间里扣除
    updateTask(id, { status: 'paused' });
  }
  function resumeTask(id: string) {
    const agg = tasks.value.find(t => t.id === id);
    if (agg?.isFolderAgg) {
      // 聚合:继续所有暂停子文件
      tasks.value.forEach(c => {
        if (c.folderId === id && !c.isFolderAgg && c.status === 'paused') resumeTask(c.id);
      });
      return;
    }
    const ctrl = controllers.get(id);
    if (!ctrl) return;
    if (ctrl.pausedAt) {
      ctrl.pausedMs += Date.now() - ctrl.pausedAt;
      ctrl.pausedAt = undefined;
    }
    ctrlResume(ctrl);
    // 速度采样基线重置:恢复后重新首采,避免暂停时长被计入段时长导致速度偏低
    ctrl.speedLastTs = undefined;
    ctrl.speedLastUploaded = undefined;
    updateTask(id, { status: 'uploading' });
  }
  function retryTask(id: string) {
    const agg = tasks.value.find(t => t.id === id);
    if (agg?.isFolderAgg) {
      // 聚合:重试所有失败子文件
      tasks.value.forEach(c => {
        if (c.folderId === id && !c.isFolderAgg && c.status === 'error') retryTask(c.id);
      });
      return;
    }
    const ctrl = controllers.get(id);
    if (!ctrl) return;
    ctrl.cancelled = false;
    ctrl.paused = false;
    // eslint-disable-next-line @typescript-eslint/no-floating-promises
    runUpload(id, ctrl);
  }
  async function cancelTask(id: string) {
    const agg = tasks.value.find(t => t.id === id);
    if (agg?.isFolderAgg) {
      // 聚合:取消所有子文件,再移除聚合条目
      const children = tasks.value.filter(c => c.folderId === id && !c.isFolderAgg);
      await Promise.all(children.map(c => cancelTask(c.id)));
      cleanupFolderState(id);
      tasks.value = tasks.value.filter(x => x.id !== id);
      return;
    }
    const ctrl = controllers.get(id);
    if (ctrl) {
      ctrl.cancelled = true;
      ctrlResume(ctrl); // 唤醒可能 paused 的 worker,使其退出
      if (ctrl.rejectSleep) ctrl.rejectSleep(new CancelledSentinel()); // 打断退避 sleep,无需等完
      if (ctrl.identifier) {
        try {
          await fetchCancelUpload(ctrl.identifier);
        } catch {
          /* 最佳努力取消,忽略 */
        }
      }
      controllers.delete(id);
    }
    // 排队任务(未启动)从队列移除
    const qIdx = uploadQueue.findIndex(q => q.id === id);
    if (qIdx >= 0) uploadQueue.splice(qIdx, 1);
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

  /** 清除已完成/失败的任务(聚合条目连同其子文件一并清除) */
  function clearFinished() {
    const removeIds = new Set<string>();
    for (const t of tasks.value) {
      if (t.status === 'success' || t.status === 'error') {
        removeIds.add(t.id);
        if (t.isFolderAgg) {
          // 聚合终态:连子文件一起清,并释放聚合瞬时状态
          cleanupFolderState(t.id);
          for (const c of tasks.value) {
            if (c.folderId === t.id && !c.isFolderAgg) removeIds.add(c.id);
          }
        }
      }
    }
    tasks.value = tasks.value.filter(t => !removeIds.has(t.id));
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
    hidePanel,
    togglePanel,
    reloadConfig
  };
}
