/// <reference lib="webworker" />
import SparkMD5 from 'spark-md5';

/**
 * 网盘分片 MD5 计算 Web Worker(M2)。
 *
 * 主线程 chunkMd5(blob) 调度到此 Worker:接收 {id, blob},算 MD5 后回 {ok, id, md5}。
 * 大分片(最大 50MB)MD5 计算移到后台,避免阻塞主线程 UI。主线程维护 Worker 池轮询分配。
 *
 * 关键:请求/响应按 id 关联。Worker 池被多文件并行上传共享,同一 Worker 上会排队多个分片,
 * 没有 id 的话,后到的调用的监听器会收到前一个分片的结果,返回错误 MD5 → 后端"分片 N 校验失败"。
 */
const ctx = self as unknown as DedicatedWorkerGlobalScope;

ctx.onmessage = async (e: MessageEvent) => {
  const { id, blob } = e.data as { id: number; blob: Blob };
  try {
    const buf = await blob.arrayBuffer();
    const spark = new SparkMD5.ArrayBuffer();
    spark.append(buf);
    ctx.postMessage({ ok: true, id, md5: spark.end() });
  } catch (err) {
    ctx.postMessage({ ok: false, id, error: (err as Error).message });
  }
};
