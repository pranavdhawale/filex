const API_BASE = import.meta.env.VITE_API_URL || "";

export async function initUpload(data: {
  filename: string;
  size: number;
  ttl_seconds: number;
  content_type: string;
  encrypted_fek: string;
  salt: string;
  chunk_size: number;
  total_chunks: number;
}) {
  const res = await fetch(`${API_BASE}/api/v1/files/init`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error(`Init failed (${res.status})`);
  return res.json();
}

export async function completeUpload(fileId: string, chunkHashes: string[]) {
  const res = await fetch(`${API_BASE}/api/v1/files/complete`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ file_id: fileId, chunk_hashes: chunkHashes }),
  });
  if (!res.ok) throw new Error(`Complete failed (${res.status})`);
  return res.json();
}

export async function accessFile(slug: string) {
  const res = await fetch(`${API_BASE}/api/v1/files/${slug}/access`, {
    method: "POST",
  });
  if (!res.ok) {
    if (res.status === 404) throw new Error("FILE_NOT_FOUND");
    if (res.status === 410) throw new Error("DOWNLOAD_LIMIT");
    throw new Error(`Access failed (${res.status})`);
  }
  return res.json();
}

export async function createShare(data: {
  file_ids: string[];
  ttl_seconds: number;
  max_downloads: number;
}) {
  const res = await fetch(`${API_BASE}/api/v1/shares`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error(`Create share failed (${res.status})`);
  return res.json();
}

export async function getShare(shareSlug: string) {
  const res = await fetch(`${API_BASE}/api/v1/shares/${shareSlug}`);
  if (!res.ok) {
    if (res.status === 404) throw new Error("SHARE_NOT_FOUND");
    if (res.status === 410) throw new Error("DOWNLOAD_LIMIT");
    throw new Error(`Get share failed (${res.status})`);
  }
  return res.json();
}