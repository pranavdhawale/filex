/**
 * upload.ts — Streaming chunked upload engine.
 * Reads file in chunks, encrypts each, uploads in parallel (max 5), retries.
 */

import { generateFEK, encryptChunk, bytesToBase64, wrapFEKWithPassphrase } from "./crypto";
import { initUpload, completeUpload } from "./api";
import type { UploadOptions, UploadResult, PartInfo } from "@/types";

const CHUNK_SIZE = 10 * 1024 * 1024; // 10 MB
const MAX_CONCURRENCY = 5;
const MAX_RETRIES = 3;

async function uploadChunkWithRetry(
  url: string,
  encryptedChunk: Uint8Array,
  retries = MAX_RETRIES
): Promise<string> {
  for (let attempt = 1; attempt <= retries; attempt++) {
    try {
      const body: ArrayBuffer = (() => {
        const buf = new ArrayBuffer(encryptedChunk.byteLength);
        new Uint8Array(buf).set(encryptedChunk);
        return buf;
      })();
      const res = await fetch(url, {
        method: "PUT",
        body,
        headers: { "Content-Type": "application/octet-stream" },
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const etag = res.headers.get("ETag") ?? "";
      return etag.replace(/"/g, "");
    } catch (err) {
      if (attempt === retries) throw err;
      await new Promise((r) => setTimeout(r, attempt * 1000));
    }
  }
  throw new Error("Upload failed after retries");
}

export async function startUpload(opts: UploadOptions): Promise<UploadResult> {
  const { file, ttlDays, encryptionMode, passphrase, onProgress } = opts;

  const fek = generateFEK();
  const totalChunks = Math.ceil(file.size / CHUNK_SIZE);

  // Init upload
  const init = await initUpload(file.size, ttlDays, encryptionMode);
  const serverChunkSize = init.chunk_size;
  const serverTotalChunks = init.total_chunks;

  const parts: PartInfo[] = new Array(serverTotalChunks);
  let chunksDone = 0;
  let bytesUploaded = 0;
  const startTime = Date.now();

  // Semaphore: limit to MAX_CONCURRENCY parallel uploads
  const semaphore = new Array(MAX_CONCURRENCY).fill(Promise.resolve());
  let semIdx = 0;

  const tasks: Promise<void>[] = [];

  for (let i = 0; i < serverTotalChunks; i++) {
    const chunkIndex = i; // 0-based
    const partNumber = i + 1; // 1-based S3 part number

    const sliceStart = chunkIndex * serverChunkSize;
    const sliceEnd = Math.min(sliceStart + serverChunkSize, file.size);
    const slice = file.slice(sliceStart, sliceEnd);

    const slot = semIdx % MAX_CONCURRENCY;
    semIdx++;

    const task = (semaphore[slot] = semaphore[slot].then(async () => {
      onProgress({
        phase: "encrypting",
        chunksDone,
        totalChunks: serverTotalChunks,
        bytesUploaded,
        totalBytes: file.size,
        speedMBps: 0,
        etaSecs: 0,
      });

      const plaintext = await slice.arrayBuffer();
      const encrypted = await encryptChunk(plaintext, fek, partNumber);

      const url = init.presigned_urls[chunkIndex];
      const etag = await uploadChunkWithRetry(url, encrypted);

      parts[chunkIndex] = { part_number: partNumber, etag };
      chunksDone++;
      bytesUploaded += sliceEnd - sliceStart;

      const elapsed = (Date.now() - startTime) / 1000;
      const speed = bytesUploaded / elapsed / (1024 * 1024);
      const remaining = file.size - bytesUploaded;
      const eta = speed > 0 ? remaining / (speed * 1024 * 1024) : 0;

      onProgress({
        phase: "uploading",
        chunksDone,
        totalChunks: serverTotalChunks,
        bytesUploaded,
        totalBytes: file.size,
        speedMBps: speed,
        etaSecs: eta,
      });
    }));

    tasks.push(task);
  }

  await Promise.all(tasks);

  onProgress({
    phase: "completing",
    chunksDone,
    totalChunks: serverTotalChunks,
    bytesUploaded: file.size,
    totalBytes: file.size,
    speedMBps: 0,
    etaSecs: 0,
  });

  // Prepare FEK for server
  let plainFEK = "";
  let encryptedFEK = "";

  if (encryptionMode === "anonymous") {
    plainFEK = bytesToBase64(fek);
  } else {
    if (!passphrase) throw new Error("Passphrase required for master mode");
    encryptedFEK = await wrapFEKWithPassphrase(fek, passphrase);
  }

  await completeUpload(
    init.file_id,
    parts,
    encryptionMode,
    plainFEK,
    encryptedFEK
  );

  const expiresAt = new Date();
  expiresAt.setDate(expiresAt.getDate() + ttlDays);

  return {
    fileId: init.file_id,
    shareUrl: `${window.location.origin}/f/${init.file_id}`,
    expiresAt,
  };
}
