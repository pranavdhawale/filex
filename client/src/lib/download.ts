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
  phase: "decrypting" | "saving";
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

  // Check FSA support before fetching — avoids double fetch
  const fsaSupported = "showSaveFilePicker" in window;

  if (fsaSupported) {
    await downloadWithFSA(meta.downloadUrl, fekBytes, slug, encryptedChunkSize, filename, onProgress);
  } else {
    await downloadWithBlobFallback(meta.downloadUrl, fekBytes, slug, encryptedChunkSize, filename, onProgress);
  }
}

async function downloadWithFSA(
  url: string,
  fekBytes: Uint8Array,
  fileId: string,
  encryptedChunkSize: number,
  filename: string,
  onProgress?: (p: DownloadProgress) => void
): Promise<void> {
  // Get save picker before fetching — user dialog up front
  const ext = filename.includes(".") ? "." + filename.split(".").pop()!.toLowerCase() : "";
  let handle: FileSystemFileHandle;
  try {
    handle = await (window as any).showSaveFilePicker({
      suggestedName: filename,
      types: [{ description: "File", accept: { "application/octet-stream": ext ? [ext] : [] } }],
    });
  } catch (err: any) {
    if (err?.name === "AbortError") throw err;
    // FSA failed despite support — fall back to blob
    await downloadWithBlobFallback(url, fekBytes, fileId, encryptedChunkSize, filename, onProgress);
    return;
  }

  const writable = await handle.createWritable();

  try {
    const res = await fetch(url);
    if (!res.ok || !res.body) throw new Error(`Fetch failed: ${res.status}`);

    const totalBytes = parseInt(res.headers.get("Content-Length") || "0", 10);
    onProgress?.({ phase: "decrypting", receivedBytes: 0, totalBytes, speedMBps: 0, etaSecs: 0 });

    const decrypted = res.body
      .pipeThrough(createProgressTransformer(totalBytes, onProgress))
      .pipeThrough(createDecryptTransformer(fekBytes, fileId, encryptedChunkSize));

    onProgress?.({ phase: "saving", receivedBytes: totalBytes, totalBytes, speedMBps: 0, etaSecs: 0 });
    await decrypted.pipeTo(writable);
  } catch (err) {
    // Abort the writable on error so we don't leave a partial file
    await writable.abort().catch(() => {});
    throw err;
  }
}

async function downloadWithBlobFallback(
  url: string,
  fekBytes: Uint8Array,
  fileId: string,
  encryptedChunkSize: number,
  filename: string,
  onProgress?: (p: DownloadProgress) => void
): Promise<void> {
  const res = await fetch(url);
  if (!res.ok || !res.body) throw new Error(`Fetch failed: ${res.status}`);

  const totalBytes = parseInt(res.headers.get("Content-Length") || "0", 10);
  onProgress?.({ phase: "decrypting", receivedBytes: 0, totalBytes, speedMBps: 0, etaSecs: 0 });

  const decrypted = res.body
    .pipeThrough(createProgressTransformer(totalBytes, onProgress))
    .pipeThrough(createDecryptTransformer(fekBytes, fileId, encryptedChunkSize));

  // Collect decrypted chunks into a single blob
  const chunks: Uint8Array[] = [];
  const reader = decrypted.getReader();
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
  }

  onProgress?.({ phase: "saving", receivedBytes: totalBytes, totalBytes, speedMBps: 0, etaSecs: 0 });

  const blob = new Blob(chunks as BlobPart[], { type: "application/octet-stream" });
  const blobUrl = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = blobUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  setTimeout(() => URL.revokeObjectURL(blobUrl), 5000);
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
        phase: "decrypting",
        receivedBytes: received,
        totalBytes,
        speedMBps: speed,
        etaSecs: speed > 0 ? remaining / (speed * 1024 * 1024) : 0,
      });
      controller.enqueue(chunk);
    },
  });
}