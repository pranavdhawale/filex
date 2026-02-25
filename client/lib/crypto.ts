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

/**
 * createDecryptTransformer — returns a TransformStream that:
 * - Accepts a raw stream of encrypted bytes (ReadableStream<Uint8Array>)
 * - Buffers incoming bytes until a full encrypted chunk is available
 * - Decrypts each chunk with AES-256-GCM as it arrives (streaming pipeline)
 * - Outputs decrypted plaintext Uint8Array chunks
 *
 * Encrypted chunk layout (per chunk):
 *   [ IV (12 bytes) | ciphertext (CHUNK_SIZE bytes) | GCM tag (16 bytes) ]
 *   = CHUNK_SIZE + 28 bytes total
 *
 * The last chunk may be smaller (partial file). flush() handles it.
 */
export function createDecryptTransformer(
  fekBytes: Uint8Array,
  encryptedChunkSize: number // e.g. 10*1024*1024 + 12 + 16
): TransformStream<Uint8Array, Uint8Array> {
  // Internal rolling buffer of bytes not yet decrypted
  let buffer = new Uint8Array(0);

  const appendToBuffer = (incoming: Uint8Array) => {
    const merged = new Uint8Array(buffer.length + incoming.length);
    merged.set(buffer, 0);
    merged.set(incoming, buffer.length);
    buffer = merged;
  };

  const decryptOne = async (chunk: Uint8Array): Promise<Uint8Array> => {
    const iv = chunk.slice(0, IV_LENGTH);
    const ciphertext = chunk.slice(IV_LENGTH);
    const key = await importFEK(fekBytes, "decrypt");
    const plain = await crypto.subtle.decrypt(
      { name: ALGORITHM, iv: toBuffer(iv) },
      key,
      toBuffer(ciphertext)
    );
    return new Uint8Array(plain);
  };

  return new TransformStream<Uint8Array, Uint8Array>({
    async transform(incoming, controller) {
      appendToBuffer(incoming);

      // Drain all complete encrypted chunks from the buffer
      while (buffer.length >= encryptedChunkSize) {
        const encChunk = buffer.slice(0, encryptedChunkSize);
        buffer = buffer.slice(encryptedChunkSize);
        try {
          const plain = await decryptOne(encChunk);
          controller.enqueue(plain);
        } catch (err) {
          controller.error(err);
          return;
        }
      }
    },

    async flush(controller) {
      // Decrypt the final (potentially smaller) chunk
      if (buffer.length > 0) {
        try {
          const plain = await decryptOne(buffer);
          controller.enqueue(plain);
        } catch (err) {
          controller.error(err);
        }
      }
    },
  });
}
