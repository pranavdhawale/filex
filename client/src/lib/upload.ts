import { generateFEK, encryptChunk, bytesToBase64, wrapFEK } from "./crypto";
import { initUpload, completeUpload } from "./api";

const CHUNK_SIZE = 10 * 1024 * 1024;

export interface UploadOptions {
  file: File;
  ttlSeconds: number;
  passphrase: string;
  onProgress?: (progress: UploadProgress) => void;
}

export interface UploadProgress {
  phase: "encrypting" | "uploading" | "completing";
  chunksDone: number;
  totalChunks: number;
  bytesUploaded: number;
  totalBytes: number;
  speedMBps: number;
  etaSecs: number;
}

export interface UploadResult {
  fileId: string;
  slug: string;
  shareUrl: string;
  expiresAt: Date;
}

export async function startUpload(opts: UploadOptions): Promise<UploadResult> {
  const { file, ttlSeconds, passphrase, onProgress } = opts;
  const fek = generateFEK();
  const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
  const salt = new Uint8Array(32);
  crypto.getRandomValues(salt);

  // Derive wrapping key and wrap FEK
  const encryptedFEK = await wrapFEK(fek, passphrase, salt);

  // Initialize upload on server
  const init = await initUpload({
    filename: file.name,
    size: file.size,
    ttl_seconds: ttlSeconds,
    content_type: file.type || "application/octet-stream",
    encrypted_fek: encryptedFEK,
    salt: bytesToBase64(salt),
    chunk_size: CHUNK_SIZE,
    total_chunks: totalChunks,
  });

  // Upload encrypted chunks via multipart
  const chunkHashes: string[] = [];
  let chunksDone = 0;
  let bytesUploaded = 0;
  const startTime = Date.now();

  for (let i = 0; i < totalChunks; i++) {
    const sliceStart = i * CHUNK_SIZE;
    const sliceEnd = Math.min(sliceStart + CHUNK_SIZE, file.size);
    const slice = file.slice(sliceStart, sliceEnd);

    // Encrypt chunk
    onProgress?.({
      phase: "encrypting",
      chunksDone,
      totalChunks,
      bytesUploaded,
      totalBytes: file.size,
      speedMBps: 0,
      etaSecs: 0,
    });

    const plaintext = await slice.arrayBuffer();
    const encrypted = await encryptChunk(plaintext, fek, init.slug, i);

    // Compute SHA-256 hash of plaintext chunk for integrity
    const hashBuffer = await crypto.subtle.digest("SHA-256", plaintext);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashHex = hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
    chunkHashes.push(hashHex);

    // Upload chunk
    const partNumber = i + 1;
    const url = new URL(init.upload_url, window.location.origin);
    url.searchParams.set("file_id", init.file_id);
    url.searchParams.set("upload_id", init.upload_id);
    url.searchParams.set("part_number", partNumber.toString());

    const uploadRes = await fetch(url.toString(), {
      method: "POST",
      body: encrypted,
      headers: { "Content-Type": "application/octet-stream" },
    });

    if (!uploadRes.ok) {
      throw new Error(`Chunk upload failed (${uploadRes.status})`);
    }

    chunksDone++;
    bytesUploaded += sliceEnd - sliceStart;

    const elapsed = (Date.now() - startTime) / 1000;
    const speed = bytesUploaded / elapsed / (1024 * 1024);

    onProgress?.({
      phase: "uploading",
      chunksDone,
      totalChunks,
      bytesUploaded,
      totalBytes: file.size,
      speedMBps: speed,
      etaSecs: speed > 0 ? (file.size - bytesUploaded) / (speed * 1024 * 1024) : 0,
    });
  }

  // Complete upload
  onProgress?.({
    phase: "completing",
    chunksDone: totalChunks,
    totalChunks,
    bytesUploaded: file.size,
    totalBytes: file.size,
    speedMBps: 0,
    etaSecs: 0,
  });

  const complete = await completeUpload(init.file_id, chunkHashes);
  const expiresAt = new Date();
  expiresAt.setSeconds(expiresAt.getSeconds() + ttlSeconds);

  return {
    fileId: init.file_id,
    slug: complete.slug,
    shareUrl: `${window.location.origin}/f/${complete.slug}`,
    expiresAt,
  };
}