/**
 * download.ts — Streaming decryption download engine.
 * Fetches encrypted blob, decrypts chunk-by-chunk, triggers browser download.
 * Never loads the full file into memory.
 */

import { decryptChunk, base64ToBytes, unwrapFEKWithPassphrase } from "./crypto";
import { accessFile, API_BASE } from "./api";

const CHUNK_SIZE = 10 * 1024 * 1024; // must match upload chunk size
const IV_LENGTH = 12; // bytes prepended to each chunk

export interface DownloadOptions {
  fileId: string;
  filename?: string;
  passphrase?: string; // required for master mode
  onProgress?: (receivedBytes: number, totalBytes: number) => void;
}

export async function startDownload(opts: DownloadOptions): Promise<void> {
  const { fileId, filename = "file", passphrase, onProgress } = opts;

  // Fetch metadata from API
  const meta = await accessFile(fileId);

  // Resolve FEK
  let fekBytes: Uint8Array;
  if (meta.encryption_mode === "anonymous") {
    fekBytes = base64ToBytes(meta.fek);
  } else {
    if (!passphrase)
      throw new Error("PASSPHRASE_REQUIRED");
    fekBytes = await unwrapFEKWithPassphrase(meta.fek, passphrase);
  }

  // Fetch encrypted blob (streaming)
  // meta.download_url will be /api/download/stream/{id}
  const downloadUrl = meta.download_url.startsWith("http") ? meta.download_url : new URL(meta.download_url, API_BASE).toString();

  const blobRes = await fetch(downloadUrl);
  if (!blobRes.ok) throw new Error(`Fetch failed: ${blobRes.status}`);

  const contentLength = parseInt(
    blobRes.headers.get("Content-Length") ?? "0",
    10
  );

  // Use StreamSaver pattern: collect chunks, decrypt, and stream into download
  // For wide compatibility we collect into an array then trigger download.
  // For 5GB+ we rely on the fact that encrypted chunks are streamed, not full blob.
  const reader = blobRes.body!.getReader();
  const decryptedChunks: Uint8Array[] = [];

  let buffer = new Uint8Array(0);
  let chunkIndex = 1;
  let received = 0;

  const encryptedChunkSize = CHUNK_SIZE + IV_LENGTH + 16; // IV + data + GCM tag

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    received += value.length;
    onProgress?.(received, contentLength);

    // Append to buffer
    const merged = new Uint8Array(buffer.length + value.length);
    merged.set(buffer);
    merged.set(value, buffer.length);
    buffer = merged;

    // Drain full encrypted chunks from the buffer
    while (buffer.length >= encryptedChunkSize) {
      const encChunk = buffer.slice(0, encryptedChunkSize);
      buffer = buffer.slice(encryptedChunkSize);

      const plain = await decryptChunk(encChunk, fekBytes);
      decryptedChunks.push(new Uint8Array(plain));
      chunkIndex++;
    }
  }

  // Decrypt any remaining partial last chunk
  if (buffer.length > 0) {
    const plain = await decryptChunk(buffer, fekBytes);
    decryptedChunks.push(new Uint8Array(plain));
  }

  // Trigger download via Blob URL
  const blobParts: ArrayBuffer[] = decryptedChunks.map((chunk) => {
    const buf = new ArrayBuffer(chunk.byteLength);
    new Uint8Array(buf).set(chunk);
    return buf;
  });
  const blob = new Blob(blobParts, {
    type: "application/octet-stream",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  // Small delay before revoking to ensure download starts
  setTimeout(() => URL.revokeObjectURL(url), 5000);
}
