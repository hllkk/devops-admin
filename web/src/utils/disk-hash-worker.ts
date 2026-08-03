/// <reference lib="webworker" />
import SparkMD5 from 'spark-md5';

/**
 * 网盘哈希计算 Web Worker。
 *
 * 两种任务(按 data.kind 路由):
 * - chunk: {id, kind:'chunk', blob} → {ok, id, md5}           分片 MD5(后端 SaveChunk 校验)
 * - sample:{id, kind:'sample', file} → {ok, id, quickHash, strongHash, midHash}  秒传采样指纹
 *
 * 采样哈希(quick=MD5采样 / strong=SHA-256采样 / mid=中间盲区MD5)原先在主线程算,大文件 hashing
 * 阶段阻塞 UI。移入 Worker 与 chunkMd5 共用池,根治卡顿。
 *
 * 关键:请求/响应按 id 关联。Worker 池被多文件并行上传共享,同一 Worker 上会排队多个任务,
 * 没有 id 的话后到的调用会收到前一个的结果(chunkMd5 曾因此返回错误 MD5 → 后端"分片 N 校验失败")。
 */
const ctx = self as unknown as DedicatedWorkerGlobalScope;
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
 * 秒传指纹:quickHash(MD5 采样) + strongHash(SHA-256 采样,防碰撞) + midHash(中间盲区 MD5)。
 * crypto.subtle 不可用(非安全上下文)时 strongHash 留空,后端按 quickHash+strongHash 查询仍自洽。
 * midHash 仅 >4MB 文件有中间盲区(首尾各 2MB 之外),<=4MB 首尾已全覆盖,置空跳过。
 */
async function computeSampleHashes(
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

/**
 * 分片 MD5 流式计算:按 STEP 步长逐块 arrayBuffer + SparkMD5 增量 append。
 * SparkMD5.ArrayBuffer 内部按 64 字节块增量更新 state、只保留不足一块的尾部,
 * 故内存峰值 ≈ STEP 而非整片——原实现 await blob.arrayBuffer() 把最大 50MB 整片一次性读入,
 * 多分片并发(Worker 池 × 分片并发)时 Worker 内存压力大,移动端/低端机易触发标签页崩溃。
 * sample/mid 采样量小(≤4MB)不需流式,仅 chunk 走此路径。
 */
async function chunkMd5Stream(blob: Blob): Promise<string> {
  const spark = new SparkMD5.ArrayBuffer();
  const STEP = 1024 * 1024; // 1MB 步长:内存峰值 ~1MB,平衡循环次数与内存
  const total = blob.size;
  for (let off = 0; off < total; off += STEP) {
    spark.append(await blob.slice(off, Math.min(off + STEP, total)).arrayBuffer());
  }
  return spark.end();
}

ctx.onmessage = async (e: MessageEvent) => {
  const data = e.data as { id: number; kind?: 'chunk' | 'sample'; blob?: Blob; file?: File };
  const { id, kind = 'chunk' } = data;
  try {
    if (kind === 'sample') {
      if (!data.file) throw new Error('sample 任务缺少 file');
      const { quickHash, strongHash, midHash } = await computeSampleHashes(data.file);
      ctx.postMessage({ ok: true, id, kind: 'sample', quickHash, strongHash, midHash });
    } else {
      if (!data.blob) throw new Error('chunk 任务缺少 blob');
      ctx.postMessage({ ok: true, id, md5: await chunkMd5Stream(data.blob) });
    }
  } catch (err) {
    ctx.postMessage({ ok: false, id, error: (err as Error).message });
  }
};
