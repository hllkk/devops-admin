import SparkMD5 from 'spark-md5';

/**
 * 网盘上传哈希工具(对齐后端 disk_upload:quickHash=MD5 采样、strongHash=SHA-256 采样、chunkHash=分片 MD5)。
 * 与后端 service/disk 的秒传(quick_hash+strong_hash 命中)、SaveChunk(md5 校验)契约一致。
 */

const SAMPLE = 2 * 1024 * 1024; // 2MB 采样

/** 构造采样缓冲:首 2MB + size(8 字节小端) + 尾 2MB(>2MB 时)。固定写入 size 防碰撞。 */
async function buildSampleBuffer(file: File): Promise<ArrayBuffer> {
  const parts: BlobPart[] = [file.slice(0, Math.min(SAMPLE, file.size))];
  const sizeBuf = new ArrayBuffer(8);
  new DataView(sizeBuf).setBigUint64(0, BigInt(file.size), true);
  parts.push(sizeBuf);
  if (file.size > SAMPLE) {
    parts.push(file.slice(Math.max(SAMPLE, file.size - SAMPLE)));
  }
  return new Blob(parts).arrayBuffer();
}

/** bytes → hex */
function toHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

/**
 * 计算秒传指纹:quickHash(MD5 采样) + strongHash(SHA-256 采样,防碰撞)。
 * crypto.subtle 不可用(非安全上下文)时 strongHash 留空,后端按 quickHash+strongHash 查询仍自洽。
 */
export async function computeSampleHashes(
  file: File
): Promise<{ quickHash: string; strongHash: string; midHash: string }> {
  const buf = await buildSampleBuffer(file);
  const spark = new SparkMD5.ArrayBuffer();
  spark.append(buf);
  const quickHash = spark.end();
  let strongHash = '';
  if (globalThis.crypto?.subtle) {
    try {
      const digest = await globalThis.crypto.subtle.digest('SHA-256', buf);
      strongHash = toHex(new Uint8Array(digest));
    } catch {
      strongHash = '';
    }
  }
  // 中间块 MD5(秒传二次校验,防首尾采样碰撞致内容张冠李戴):
  // 仅 >4MB 文件有中间盲区(首尾各2MB 之外的中间段),<=4MB 首尾已全覆盖,置空跳过校验。
  let midHash = '';
  if (file.size > 4 * 1024 * 1024) {
    const midStart = Math.max(0, Math.floor(file.size / 2) - 1024 * 1024);
    const midEnd = Math.min(file.size, midStart + 2 * 1024 * 1024);
    const midBuf = await file.slice(midStart, midEnd).arrayBuffer();
    const midSpark = new SparkMD5.ArrayBuffer();
    midSpark.append(midBuf);
    midHash = midSpark.end();
  }
  return { quickHash, strongHash, midHash };
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
  resolve: (md5: string) => void;
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
      const data = e.data as { ok: boolean; id: number; md5?: string; error?: string };
      const job = pendingJobs.get(data.id);
      if (!job) return; // 找不到 id:重复响应或调用方已放弃,丢弃
      pendingJobs.delete(data.id);
      if (data.ok) job.resolve(data.md5 as string);
      else job.reject(new Error(data.error || 'hash worker 失败'));
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
    pendingJobs.set(id, { resolve, reject });
    worker.postMessage({ id, blob });
  });
}

/** 动态分片大小(按文件大小,与 remote 思路一致,前端简化档)。 */
export function getChunkSize(fileSize: number): number {
  if (fileSize < 100 * 1024 * 1024) return 10 * 1024 * 1024; // <100MB → 10MB
  if (fileSize < 1024 * 1024 * 1024) return 20 * 1024 * 1024; // <1GB → 20MB
  return 50 * 1024 * 1024; // ≥1GB → 50MB
}
