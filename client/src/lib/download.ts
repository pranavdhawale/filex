import { unwrapFEK, createDecryptTransformer, base64ToBytes } from "./crypto";
import { accessFile } from "./api";

const IV_LENGTH = 12;
const GCM_TAG_LENGTH = 16;

export interface DownloadOptions {
  slug: string;
  passphrase: string;
  onProgress?: (progress: DownloadProgress) => void;
}

export interface DownloadProgress {
  phase: "fetching" | "decrypting" | "saving";
  receivedBytes: number;
  totalBytes: number;
  speedMBps: number;
  etaSecs: number;
}

export async function startDownload(opts: DownloadOptions): Promise<void> {
  const { slug, passphrase, onProgress } = opts;

  const meta = await accessFile(slug);
  const filename = meta.filename || slug;
  const fekBytes = await unwrapFEK(meta.encryptedFek, passphrase, base64ToBytes(meta.salt));
  const encryptedChunkSize = meta.chunkSize + IV_LENGTH + GCM_TAG_LENGTH;

  // Fetch encrypted data from presigned URL
  const res = await fetch(meta.downloadUrl);
  if (!res.ok || !res.body) throw new Error(`Fetch failed: ${res.status}`);

  const totalBytes = parseInt(res.headers.get("Content-Length") || "0", 10);

  onProgress?.({ phase: "fetching", receivedBytes: 0, totalBytes, speedMBps: 0, etaSecs: 0 });

  // Build decrypt pipeline
  const progressStream = createProgressTransformer(totalBytes, onProgress);
  const decryptedStream = res.body
    .pipeThrough(progressStream)
    .pipeThrough(createDecryptTransformer(fekBytes, slug, encryptedChunkSize));

  // Save to disk
  const usedFSA = await saveWithFileSystemAPI(decryptedStream, filename, onProgress, totalBytes);
  if (!usedFSA) {
    // Re-fetch for fallback (stream consumed)
    const res2 = await fetch(meta.downloadUrl);
    if (!res2.ok || !res2.body) throw new Error("Fallback fetch failed");
    const ps2 = createProgressTransformer(totalBytes, onProgress);
    const ds2 = res2.body
      .pipeThrough(ps2)
      .pipeThrough(createDecryptTransformer(fekBytes, slug, encryptedChunkSize));
    await saveWithBlobFallback(ds2, filename, onProgress, totalBytes);
  }
}

function createProgressTransformer(
  totalBytes: number,
  onProgress?: (p: DownloadProgress) => void
): TransformStream<Uint8Array, Uint8Array> {
  let received = 0;
  const start = Date.now();
  return new TransformStream({
    transform(chunk, controller) {
      received += chunk.length;
      const elapsed = (Date.now() - start) / 1000 || 0.001;
      const speed = received / elapsed / (1024 * 1024);
      const remaining = totalBytes - received;
      onProgress?.({
        phase: "fetching",
        receivedBytes: received,
        totalBytes,
        speedMBps: speed,
        etaSecs: speed > 0 ? remaining / (speed * 1024 * 1024) : 0,
      });
      controller.enqueue(chunk);
    },
  });
}

async function saveWithFileSystemAPI(
  stream: ReadableStream<Uint8Array>,
  filename: string,
  onProgress?: (p: DownloadProgress) => void,
  totalBytes?: number
): Promise<boolean> {
  if (!("showSaveFilePicker" in window)) return false;
  try {
    const ext = filename.includes(".") ? "." + filename.split(".").pop()!.toLowerCase() : "";
    const handle = await (window as any).showSaveFilePicker({
      suggestedName: filename,
      types: [{ description: "File", accept: { "application/octet-stream": ext ? [ext] : [] } }],
    });
    const writable = await handle.createWritable();
    await stream.pipeTo(writable);
    return true;
  } catch (err: any) {
    if (err?.name === "AbortError") throw err;
    return false;
  }
}

async function saveWithBlobFallback(
  stream: ReadableStream<Uint8Array>,
  filename: string,
  onProgress?: (p: DownloadProgress) => void,
  totalBytes?: number
): Promise<void> {
  const chunks: Uint8Array[] = [];
  const reader = stream.getReader();
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
  }
  const blob = new Blob(chunks, { type: "application/octet-stream" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  setTimeout(() => URL.revokeObjectURL(url), 5000);
}