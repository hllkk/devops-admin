/**
 * 网盘上传哈希工具(对齐后端 disk_upload:quickHash=MD5 采样、strongHash=SHA-256 采样、chunkHash=分片 MD5)。
 * 与后端 service/disk 的秒传(quick_hash+strong_hash 命中)、SaveChunk(md5 校验)契约一致。
 *
 * 所有哈希(chunkMd5 / 秒传采样指纹)均调度到 disk-hash-worker.ts 的 Web Worker 池后台计算,
 * 主线程不直接算 MD5/SHA-256,避免大文件 hashing 阶段阻塞 UI。
 */

/**
 * 计算秒传指纹(quickHash/strongHash/midHash):投递 File 到 Worker 池后台算,不阻塞主线程。
 * 哈希实现(采样缓冲构造 + MD5/SHA-256)在 disk-hash-worker.ts,本函数仅投递 + 按 id 收结果。
 */
export function computeSampleHashes(
  file: File
): Promise<{ quickHash: string; strongHash: string; midHash: string }> {
  const pool = ensureWorkerPool();
  const worker = pool[workerIdx % pool.length]!;
  workerIdx += 1;
  const id = nextReqId++;
  return new Promise<{ quickHash: string; strongHash: string; midHash: string }>((resolve, reject) => {
    pendingJobs.set(id, { resolve: resolve as (v: unknown) => void, reject });
    worker.postMessage({ id, kind: 'sample', file });
  });
}

// Web Worker 池(M2):chunkMd5 走后台计算,避免大分片(最大 50MB)全量入内存算 MD5 阻塞主线程 UI。
// 池懒加载,大小取 navigator.hardwareConcurrency(上限 8),轮询分配任务。
//
// 关键:请求/响应用自增 id 关联。多文件并行上传时同一 Worker 上会排队多个分片,必须靠 id 把响应
// 路由回正确的调用者。旧实现用 addEventListener 监听"该 Worker 的任意消息",并发时后到的调用会被
// 前一个分片的结果 resolve,返回错误 MD5 → 后端"分片 N 校验失败"。改用每 Worker 单个 onmessage + Map 路由。
let workerPool: Worker[] | null = null;
let workerIdx = 0;
let nextReqId = 1;
interface PendingJob {
  resolve: (value: unknown) => void;
  reject: (err: Error) => void;
}
const pendingJobs = new Map<number, PendingJob>();

function ensureWorkerPool(): Worker[] {
  if (workerPool) return workerPool;
  const cores = Math.min(navigator.hardwareConcurrency || 4, 8);
  const size = Math.max(2, cores);
  workerPool = Array.from({ length: size }, () => {
    const w = new Worker(new URL('./disk-hash-worker.ts', import.meta.url), { type: 'module' });
    // 每 Worker 单个 onmessage:按 id 取出对应调用的 resolve/reject,响应与到达顺序无关。
    w.onmessage = (e: MessageEvent) => {
      const data = e.data as {
        ok: boolean;
        id: number;
        md5?: string;
        quickHash?: string;
        strongHash?: string;
        midHash?: string;
        error?: string;
      };
      const job = pendingJobs.get(data.id);
      if (!job) return; // 找不到 id:重复响应或调用方已放弃,丢弃
      pendingJobs.delete(data.id);
      if (!data.ok) {
        job.reject(new Error(data.error || 'hash worker 失败'));
        return;
      }
      // sample 任务回指纹对象,chunk 任务回 md5 字符串(按有无 quickHash 区分)
      if (data.quickHash !== undefined) {
        job.resolve({ quickHash: data.quickHash, strongHash: data.strongHash || '', midHash: data.midHash || '' });
      } else {
        job.resolve(data.md5 as string);
      }
    };
    return w;
  });
  return workerPool;
}

/** 计算单个分片的 MD5(后端 SaveChunk 的 chunkHash 校验用)。
 *  走 Web Worker 池(M2),不阻塞主线程;按 id 路由响应,多文件并行上传并发安全。 */
export function chunkMd5(blob: Blob): Promise<string> {
  const pool = ensureWorkerPool();
  const worker = pool[workerIdx % pool.length]!;
  workerIdx += 1;
  const id = nextReqId++;
  return new Promise<string>((resolve, reject) => {
    pendingJobs.set(id, { resolve: resolve as (v: unknown) => void, reject });
    worker.postMessage({ id, kind: 'chunk', blob });
  });
}

/** 设备内存(GB)。navigator.deviceMemory 仅部分浏览器支持,且只返回离散值(0.25/0.5/1/2/4/8)。 */
function deviceMemGB(): number {
  const m = (navigator as Navigator & { deviceMemory?: number }).deviceMemory;
  return typeof m === 'number' && m > 0 ? m : 4; // 不可用时默认按 4GB
}
/**
 * 动态分片大小:按文件大小分级,再按设备内存缩放。
 * 低内存设备(<2GB,多为移动端)整体降一档,控单片内存峰值;与 remote 设备感知思路一致。
 */
export function getChunkSize(fileSize: number): number {
  const MB = 1024 * 1024;
  const base = fileSize < 100 * MB ? 10 * MB : fileSize < 1024 * MB ? 20 * MB : 50 * MB;
  const scale = deviceMemGB() < 2 ? 0.5 : 1;
  return Math.round(base * scale);
}
