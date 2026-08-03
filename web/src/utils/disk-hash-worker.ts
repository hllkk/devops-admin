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
      const buf = await data.blob.arrayBuffer();
      const spark = new SparkMD5.ArrayBuffer();
      spark.append(buf);
      ctx.postMessage({ ok: true, id, md5: spark.end() });
    }
  } catch (err) {
    ctx.postMessage({ ok: false, id, error: (err as Error).message });
  }
};
