export interface InitUploadResponse {
  file_id: string;
  upload_id: string;
  chunk_size: number;
  total_chunks: number;
}

export interface CompleteUploadResponse {
  status: string;
  file_id: string;
}

export interface AccessFileResponse {
  download_url: string;
  fek: string; // base64 — plain FEK for anonymous, wrapped for master
  encryption_mode: "anonymous" | "master";
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
  shareUrl: string;
  expiresAt: Date;
}

export interface PartInfo {
  part_number: number;
  etag: string;
}
