export const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

import type {
  InitUploadResponse,
  CompleteUploadResponse,
  AccessFileResponse,
  EncryptionMode,
  PartInfo,
} from "@/types";

export async function initUpload(
  size: number,
  ttlSeconds: number,
  encryptionMode: EncryptionMode,
  filename: string
): Promise<InitUploadResponse> {
  const res = await fetch(`${API_BASE}/upload/init`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      size,
      ttl_seconds: ttlSeconds,
      encryption_mode: encryptionMode,
      filename,
    }),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Init failed (${res.status}): ${text}`);
  }
  return res.json();
}

export async function completeUpload(
  fileId: string,
  parts: PartInfo[],
  encryptionMode: EncryptionMode,
  plainFEK: string, // base64, only sent for anonymous mode
  encryptedFEK: string // base64, only set for master mode
): Promise<CompleteUploadResponse> {
  const res = await fetch(`${API_BASE}/upload/complete`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      file_id: fileId,
      parts,
      encryption_mode: encryptionMode,
      plain_fek: plainFEK,
      encrypted_fek: encryptedFEK,
    }),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Complete failed (${res.status}): ${text}`);
  }
  return res.json();
}

export async function accessFile(fileId: string): Promise<AccessFileResponse> {
  const res = await fetch(`${API_BASE}/f/${fileId}`);
  if (!res.ok) {
    if (res.status === 404) throw new Error("FILE_NOT_FOUND");
    if (res.status === 410) throw new Error("FILE_EXPIRED");
    const text = await res.text();
    throw new Error(`Access failed (${res.status}): ${text}`);
  }
  return res.json();
}
