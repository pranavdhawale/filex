/**
 * download.ts — Streaming decryption download engine.
 *
 * Pipeline:
 *   fetch(encryptedBlob)
 *     .body (ReadableStream<Uint8Array>)
 *     .pipeThrough(progressTransform)      — tracks bytes received, fires onProgress
 *     .pipeThrough(createDecryptTransformer) — decrypts chunk-by-chunk as bytes arrive
 *     .pipeTo(writableDestination)         — Strategy A: File System Access API (disk stream)
 *                                            Strategy B: Response blob fallback (Firefox)
 *
 * Peak RAM: ~2 encrypted chunks (~20 MB) regardless of total file size.
 * Compare to old approach: entire file in RAM (2x file size).
 */

import { base64ToBytes, unwrapFEKWithPassphrase, createDecryptTransformer } from "./crypto";
import { accessFile, API_BASE } from "./api";
import type { DownloadOptions, DownloadProgress } from "@/types";

// Must match upload chunk size exactly
const PLAIN_CHUNK_SIZE = 10 * 1024 * 1024; // 10 MB
const IV_LENGTH = 12;
const GCM_TAG_LENGTH = 16;
const ENCRYPTED_CHUNK_SIZE = PLAIN_CHUNK_SIZE + IV_LENGTH + GCM_TAG_LENGTH; // 10,485,788 bytes

/**
 * Creates a pass-through TransformStream that counts bytes and fires progress callbacks.
 */
function createProgressTransformer(
  totalBytes: number,
  onProgress: (progress: DownloadProgress) => void
): TransformStream<Uint8Array, Uint8Array> {
  let receivedBytes = 0;
  const startTime = Date.now();

  return new TransformStream<Uint8Array, Uint8Array>({
    transform(chunk, controller) {
      receivedBytes += chunk.length;
      const elapsed = (Date.now() - startTime) / 1000 || 0.001;
      const speedMBps = receivedBytes / elapsed / (1024 * 1024);
      const remaining = totalBytes - receivedBytes;
      const etaSecs = speedMBps > 0 ? remaining / (speedMBps * 1024 * 1024) : 0;

      onProgress({
        phase: "fetching",
        receivedBytes,
        totalBytes,
        speedMBps,
        etaSecs,
      });

      controller.enqueue(chunk);
    },
  });
}

/**
 * Triggers a browser "Save As" dialog using the File System Access API.
 * Returns true if the strategy succeeded, false if the API is unavailable.
 */
async function saveWithFileSystemAPI(
  stream: ReadableStream<Uint8Array>,
  filename: string,
  onProgress: ((progress: DownloadProgress) => void) | undefined,
  totalBytes: number
): Promise<boolean> {
  if (!("showSaveFilePicker" in window)) return false;

  try {
    // Prompt the OS "Save As" dialog — must be called during a user gesture
    const handle = await (window as any).showSaveFilePicker({
      suggestedName: filename,
      types: [{ description: "File", accept: { "application/octet-stream": [] } }],
    });

    const writable = await handle.createWritable();

    // Pipe decrypted stream directly to the OS file — zero buffering in JS heap
    let savedBytes = 0;
    const startTime = Date.now();

    await stream.pipeTo(
      new WritableStream({
        async write(chunk) {
          await writable.write(chunk);
          savedBytes += chunk.length;
          const elapsed = (Date.now() - startTime) / 1000 || 0.001;
          const speedMBps = savedBytes / elapsed / (1024 * 1024);
          const etaSecs = speedMBps > 0 ? (totalBytes - savedBytes) / (speedMBps * 1024 * 1024) : 0;

          onProgress?.({
            phase: "saving",
            receivedBytes: savedBytes,
            totalBytes,
            speedMBps,
            etaSecs,
          });
        },
        async close() {
          await writable.close();
        },
        async abort(reason) {
          await writable.abort(reason);
        },
      })
    );

    return true;
  } catch (err: any) {
    // User cancelled the dialog — rethrow so UI can handle it
    if (err?.name === "AbortError") throw err;
    // Other errors (e.g., API not supported in this context) → fallback
    return false;
  }
}

/**
 * Fallback: collect decrypted stream into a Blob then trigger a download link click.
 * Used for Firefox and browsers without File System Access API.
 * Decryption is still streaming/pipelined — only the final blob assembly requires memory.
 */
async function saveWithBlobFallback(
  stream: ReadableStream<Uint8Array>,
  filename: string,
  onProgress: ((progress: DownloadProgress) => void) | undefined,
  totalBytes: number
): Promise<void> {
  const chunks: Uint8Array[] = [];
  let saved = 0;
  const startTime = Date.now();

  const reader = stream.getReader();
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
    saved += value.length;
    const elapsed = (Date.now() - startTime) / 1000 || 0.001;
    const speedMBps = saved / elapsed / (1024 * 1024);
    const etaSecs = speedMBps > 0 ? (totalBytes - saved) / (speedMBps * 1024 * 1024) : 0;
    onProgress?.({ phase: "decrypting", receivedBytes: saved, totalBytes, speedMBps, etaSecs });
  }

  const blob = new Blob(
    chunks.map((c) => new Uint8Array(c)),
    { type: "application/octet-stream" }
  );
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  setTimeout(() => URL.revokeObjectURL(url), 5000);
}

/**
 * Main download entry point.
 *
 * 1. Fetches file metadata + encrypted FEK from the API.
 * 2. Resolves the plaintext FEK (anonymous: direct; master: PBKDF2+AES-KW unwrap).
 * 3. Fetches the encrypted blob as a ReadableStream.
 * 4. Pipes through progressTransform → decryptTransformer → disk (or blob fallback).
 */
export async function startDownload(opts: DownloadOptions): Promise<void> {
  const { fileId, filename = `file-${fileId.slice(0, 8)}`, passphrase, onProgress } = opts;

  // ── Step 1: Fetch metadata ────────────────────────────────────────────────
  const meta = await accessFile(fileId);

  // ── Step 2: Resolve FEK ───────────────────────────────────────────────────
  let fekBytes: Uint8Array;
  if (meta.encryption_mode === "anonymous") {
    fekBytes = base64ToBytes(meta.fek);
  } else {
    if (!passphrase) throw new Error("PASSPHRASE_REQUIRED");
    fekBytes = await unwrapFEKWithPassphrase(meta.fek, passphrase);
  }

  // ── Step 3: Fetch the encrypted stream ───────────────────────────────────
  const downloadUrl = meta.download_url.startsWith("http")
    ? meta.download_url
    : new URL(meta.download_url, API_BASE).toString();

  const blobRes = await fetch(downloadUrl);
  if (!blobRes.ok) throw new Error(`Fetch failed: ${blobRes.status}`);
  if (!blobRes.body) throw new Error("No response body");

  const totalBytes = parseInt(blobRes.headers.get("Content-Length") ?? "0", 10);

  // ── Step 4: Build the decryption pipeline ────────────────────────────────
  const noop = () => { };
  const progress = onProgress ?? noop;

  // Fire initial progress so the UI transitions immediately
  progress({ phase: "fetching", receivedBytes: 0, totalBytes, speedMBps: 0, etaSecs: 0 });

  const progressStream = onProgress
    ? blobRes.body.pipeThrough(createProgressTransformer(totalBytes, progress))
    : blobRes.body;

  const decryptedStream = progressStream.pipeThrough(
    createDecryptTransformer(fekBytes, ENCRYPTED_CHUNK_SIZE)
  );

  // ── Step 5: Write to disk ─────────────────────────────────────────────────
  // Strategy A: File System Access API — true streaming to OS disk
  // Strategy B: Blob URL fallback — Firefox / older browsers
  const usedFSA = await saveWithFileSystemAPI(decryptedStream, filename, onProgress, totalBytes);
  if (!usedFSA) {
    // decryptedStream is already consumed if saveWithFileSystemAPI partially ran and failed
    // Re-build the pipeline from the original fetch (can't reuse a consumed stream)
    // Since FSA only failed due to API unavailability (not a user cancel), we can safely fallback
    const blobRes2 = await fetch(downloadUrl);
    if (!blobRes2.ok) throw new Error(`Fetch failed on fallback: ${blobRes2.status}`);
    if (!blobRes2.body) throw new Error("No response body on fallback");

    const progressStream2 = onProgress
      ? blobRes2.body.pipeThrough(createProgressTransformer(totalBytes, progress))
      : blobRes2.body;

    const decryptedStream2 = progressStream2.pipeThrough(
      createDecryptTransformer(fekBytes, ENCRYPTED_CHUNK_SIZE)
    );

    await saveWithBlobFallback(decryptedStream2, filename, onProgress, totalBytes);
  }
}
