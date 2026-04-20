// client/src/lib/crypto.ts
const ALGORITHM = "AES-GCM";
const KEY_LENGTH = 256;
const IV_LENGTH = 12;

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

function buildIV(chunkIndex: number): ArrayBuffer {
  const buf = new ArrayBuffer(IV_LENGTH);
  new DataView(buf).setUint32(0, chunkIndex, false);
  return buf;
}

function buildAEAD(fileId: string, chunkIndex: number): ArrayBuffer {
  const enc = new TextEncoder();
  const idBytes = enc.encode(fileId);
  const idxBuf = new ArrayBuffer(4);
  new DataView(idxBuf).setUint32(0, chunkIndex, false);
  const aad = new Uint8Array(idBytes.length + 4);
  aad.set(idBytes, 0);
  aad.set(new Uint8Array(idxBuf), idBytes.length);
  return aad.buffer;
}

async function importFEK(fekBytes: Uint8Array, usage: KeyUsage): Promise<CryptoKey> {
  return crypto.subtle.importKey(
    "raw",
    toBuffer(fekBytes),
    { name: ALGORITHM, length: KEY_LENGTH },
    false,
    [usage]
  );
}

export async function encryptChunk(
  plaintext: ArrayBuffer,
  fekBytes: Uint8Array,
  fileId: string,
  chunkIndex: number
): Promise<Uint8Array> {
  const key = await importFEK(fekBytes, "encrypt");
  const iv = buildIV(chunkIndex);
  const aad = buildAEAD(fileId, chunkIndex);

  const ciphertext = await crypto.subtle.encrypt(
    { name: ALGORITHM, iv, additionalData: aad },
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
  fekBytes: Uint8Array,
  fileId: string,
  chunkIndex: number
): Promise<ArrayBuffer> {
  const iv = encrypted.slice(0, IV_LENGTH);
  const ciphertext = encrypted.slice(IV_LENGTH);
  const key = await importFEK(fekBytes, "decrypt");
  const aad = buildAEAD(fileId, chunkIndex);

  return crypto.subtle.decrypt(
    { name: ALGORITHM, iv: toBuffer(iv), additionalData: aad },
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

// FEK wrapping with AES-256-GCM (passphrase-derived key wraps the FEK)
export async function wrapFEK(
  fekBytes: Uint8Array,
  passphrase: string,
  salt: Uint8Array
): Promise<string> {
  const keyMaterial = await deriveKeyFromPassphrase(passphrase, salt);
  const wrappingKey = await crypto.subtle.importKey(
    "raw",
    keyMaterial,
    { name: ALGORITHM, length: KEY_LENGTH },
    false,
    ["wrapKey"]
  );
  const fekKey = await crypto.subtle.importKey(
    "raw",
    toBuffer(fekBytes),
    { name: ALGORITHM, length: KEY_LENGTH },
    true,
    ["encrypt", "decrypt"]
  );
  // Use AES-GCM to wrap (IV = all zeros for deterministic wrapping)
  const iv = new ArrayBuffer(IV_LENGTH);
  const wrapped = await crypto.subtle.wrapKey("raw", fekKey, wrappingKey, {
    name: ALGORITHM,
    iv,
  });
  // Prepend IV + wrapped bytes
  const result = new Uint8Array(IV_LENGTH + wrapped.byteLength);
  result.set(new Uint8Array(iv), 0);
  result.set(new Uint8Array(wrapped), IV_LENGTH);
  return bytesToBase64(result);
}

export async function unwrapFEK(
  wrappedB64: string,
  passphrase: string,
  salt: Uint8Array
): Promise<Uint8Array> {
  const keyMaterial = await deriveKeyFromPassphrase(passphrase, salt);
  const unwrappingKey = await crypto.subtle.importKey(
    "raw",
    keyMaterial,
    { name: ALGORITHM, length: KEY_LENGTH },
    false,
    ["unwrapKey"]
  );
  const wrappedData = base64ToBytes(wrappedB64);
  const iv = wrappedData.slice(0, IV_LENGTH);
  const ciphertext = wrappedData.slice(IV_LENGTH);

  const fekKey = await crypto.subtle.unwrapKey(
    "raw",
    toBuffer(ciphertext),
    unwrappingKey,
    { name: ALGORITHM, iv: toBuffer(iv) },
    { name: ALGORITHM, length: KEY_LENGTH },
    true,
    ["encrypt", "decrypt"]
  );
  const raw = await crypto.subtle.exportKey("raw", fekKey);
  return new Uint8Array(raw);
}

async function deriveKeyFromPassphrase(
  passphrase: string,
  salt: Uint8Array
): Promise<ArrayBuffer> {
  // Point argon2-browser to the WASM file in the public directory
  (globalThis as any).argon2WasmPath = "/argon2.wasm";
  const argon2 = await import("argon2-browser");
  const result = await argon2.hash({
    pass: passphrase,
    salt: Array.from(salt),
    type: (argon2 as any).ArgonType?.Argon2id ?? 2,
    mem: 47104,
    time: 1,
    parallel: 1,
    hashLen: 32,
  });
  return new Uint8Array(result.hash).buffer;
}

// Streaming decrypt transformer with FEK caching and AEAD
export function createDecryptTransformer(
  fekBytes: Uint8Array,
  fileId: string,
  encryptedChunkSize: number
): TransformStream<Uint8Array, Uint8Array> {
  let buffer = new Uint8Array(0);
  let chunkIndex = 0;
  let cachedKey: CryptoKey | null = null;

  const appendToBuffer = (incoming: Uint8Array) => {
    const merged = new Uint8Array(buffer.length + incoming.length);
    merged.set(buffer, 0);
    merged.set(incoming, buffer.length);
    buffer = merged;
  };

  return new TransformStream<Uint8Array, Uint8Array>({
    async transform(incoming, controller) {
      appendToBuffer(incoming);
      while (buffer.length >= encryptedChunkSize) {
        const encChunk = buffer.slice(0, encryptedChunkSize);
        buffer = buffer.slice(encryptedChunkSize);
        try {
          if (!cachedKey) {
            cachedKey = await importFEK(fekBytes, "decrypt");
          }
          const iv = encChunk.slice(0, IV_LENGTH);
          const ciphertext = encChunk.slice(IV_LENGTH);
          const aad = buildAEAD(fileId, chunkIndex);
          const plain = await crypto.subtle.decrypt(
            { name: ALGORITHM, iv: toBuffer(iv), additionalData: aad },
            cachedKey,
            toBuffer(ciphertext)
          );
          chunkIndex++;
          controller.enqueue(new Uint8Array(plain));
        } catch (err) {
          controller.error(err);
          return;
        }
      }
    },
    async flush(controller) {
      if (buffer.length > 0) {
        try {
          if (!cachedKey) {
            cachedKey = await importFEK(fekBytes, "decrypt");
          }
          const iv = buffer.slice(0, IV_LENGTH);
          const ciphertext = buffer.slice(IV_LENGTH);
          const aad = buildAEAD(fileId, chunkIndex);
          const plain = await crypto.subtle.decrypt(
            { name: ALGORITHM, iv: toBuffer(iv), additionalData: aad },
            cachedKey,
            toBuffer(ciphertext)
          );
          controller.enqueue(new Uint8Array(plain));
        } catch (err) {
          controller.error(err);
        }
      }
    },
  });
}