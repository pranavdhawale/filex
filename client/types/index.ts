export interface InitUploadResponse {
  file_id: string;
  upload_id: string;
  chunk_size: number;
  total_chunks: number;
}

export interface CompleteUploadResponse {
  status: string;
  file_id: string;
  slug: string; // URL-safe filename slug, e.g. "report.pdf" or "report.pdf~a1b2"
}

export interface AccessFileResponse {
  download_url: string;
  fek: string; // base64 — plain FEK for anonymous, wrapped for master
  encryption_mode: "anonymous" | "master";
  filename: string; // Original filename for the save dialog
  expires_at: string;
}

export type EncryptionMode = "anonymous" | "master";
export type TTLDays = 1 | 7 | 15;

export interface UploadOptions {
  file: File;
  ttlDays: TTLDays;
  encryptionMode: EncryptionMode;
  passphrase?: string;
  onProgress: (progress: UploadProgress) => void;
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

export interface PartInfo {
  part_number: number;
  etag: string;
}

export type DownloadPhase = "fetching" | "decrypting" | "saving";

export interface DownloadProgress {
  phase: DownloadPhase;
  receivedBytes: number;
  totalBytes: number;
  speedMBps: number;
  etaSecs: number;
}

export interface DownloadOptions {
  fileId: string;
  filename?: string;
  passphrase?: string;
  onProgress?: (progress: DownloadProgress) => void;
}
