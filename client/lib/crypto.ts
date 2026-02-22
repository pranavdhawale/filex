/**
 * crypto.ts — Client-side AES-256-GCM encryption using the Web Crypto API.
 * No sensitive data is ever logged or stored in localStorage.
 */

const ALGORITHM = "AES-GCM";
const KEY_LENGTH = 256;
const IV_LENGTH = 12; // bytes — 96-bit nonce for GCM

/** Copy a Uint8Array into a fresh, plain ArrayBuffer (avoids SharedArrayBuffer issues). */
function toBuffer(bytes: Uint8Array): ArrayBuffer {
  const buf = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(buf).set(bytes);
  return buf;
}

export function generateFEK(): Uint8Array {
  const fek = new Uint8Array(32);
  crypto.getRandomValues(fek);
  return fek;
}

async function importFEK(
  fekBytes: Uint8Array,
  usage: "encrypt" | "decrypt"
): Promise<CryptoKey> {
  return crypto.subtle.importKey(
    "raw",
    toBuffer(fekBytes),
    { name: ALGORITHM, length: KEY_LENGTH },
    false,
    [usage]
  );
}

function buildIV(chunkIndex: number): ArrayBuffer {
  const buf = new ArrayBuffer(IV_LENGTH);
  new DataView(buf).setUint32(0, chunkIndex, false);
  return buf;
}

export async function encryptChunk(
  plaintext: ArrayBuffer,
  fekBytes: Uint8Array,
  chunkIndex: number
): Promise<Uint8Array> {
  const key = await importFEK(fekBytes, "encrypt");
  const iv = buildIV(chunkIndex);
  const ciphertext = await crypto.subtle.encrypt(
    { name: ALGORITHM, iv },
    key,
    plaintext
  );
  const result = new Uint8Array(IV_LENGTH + ciphertext.byteLength);
  result.set(new Uint8Array(iv), 0);
  result.set(new Uint8Array(ciphertext), IV_LENGTH);
  return result;
}

export async function decryptChunk(
  encrypted: Uint8Array,
  fekBytes: Uint8Array
): Promise<ArrayBuffer> {
  const iv = encrypted.slice(0, IV_LENGTH);
  const ciphertext = encrypted.slice(IV_LENGTH);
  const key = await importFEK(fekBytes, "decrypt");
  return crypto.subtle.decrypt(
    { name: ALGORITHM, iv: toBuffer(iv) },
    key,
    toBuffer(ciphertext)
  );
}

export function bytesToBase64(bytes: Uint8Array): string {
  let binary = "";
  for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
}

export function base64ToBytes(b64: string): Uint8Array {
  const padded = b64.replace(/-/g, "+").replace(/_/g, "/");
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

async function deriveKeyFromPassphrase(
  passphrase: string,
  salt: Uint8Array
): Promise<CryptoKey> {
  const encoded = new TextEncoder().encode(passphrase);
  const baseKey = await crypto.subtle.importKey(
    "raw",
    toBuffer(encoded),
    { name: "PBKDF2" },
    false,
    ["deriveKey"]
  );
  return crypto.subtle.deriveKey(
    {
      name: "PBKDF2",
      salt: toBuffer(salt),
      iterations: 600_000,
      hash: "SHA-256",
    },
    baseKey,
    { name: "AES-KW", length: 256 },
    false,
    ["wrapKey", "unwrapKey"]
  );
}

export async function wrapFEKWithPassphrase(
  fekBytes: Uint8Array,
  passphrase: string
): Promise<string> {
  const salt = new Uint8Array(16);
  crypto.getRandomValues(salt);
  const wrappingKey = await deriveKeyFromPassphrase(passphrase, salt);
  const fekKey = await crypto.subtle.importKey(
    "raw",
    toBuffer(fekBytes),
    { name: ALGORITHM, length: KEY_LENGTH },
    true,
    ["encrypt", "decrypt"]
  );
  const wrapped = await crypto.subtle.wrapKey("raw", fekKey, wrappingKey, {
    name: "AES-KW",
  });
  const result = new Uint8Array(16 + wrapped.byteLength);
  result.set(salt, 0);
  result.set(new Uint8Array(wrapped), 16);
  return bytesToBase64(result);
}

export async function unwrapFEKWithPassphrase(
  wrappedB64: string,
  passphrase: string
): Promise<Uint8Array> {
  const data = base64ToBytes(wrappedB64);
  const salt = data.slice(0, 16);
  const wrapped = data.slice(16);
  const wrappingKey = await deriveKeyFromPassphrase(passphrase, salt);
  const fekKey = await crypto.subtle.unwrapKey(
    "raw",
    toBuffer(wrapped),
    wrappingKey,
    { name: "AES-KW" },
    { name: ALGORITHM, length: KEY_LENGTH },
    true,
    ["encrypt", "decrypt"]
  );
  const raw = await crypto.subtle.exportKey("raw", fekKey);
  return new Uint8Array(raw);
}
