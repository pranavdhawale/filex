// client/src/lib/crypto.worker.ts
const ctx = self as unknown as DedicatedWorkerGlobalScope;

ctx.onmessage = async (e: MessageEvent) => {
  const { type } = e.data;

  if (type === "deriveKey") {
    const { passphrase, salt } = e.data;
    try {
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

      const keyBytes = new Uint8Array(result.hash);
      const key = await crypto.subtle.importKey(
        "raw",
        keyBytes.buffer,
        { name: "AES-GCM", length: 256 },
        true,
        ["wrapKey", "unwrapKey"]
      );
      const raw = await crypto.subtle.exportKey("raw", key);
      ctx.postMessage({ type: "derivedKey", key: raw });
    } catch (err) {
      ctx.postMessage({ type: "error", message: String(err) });
    }
  }

  if (type === "encrypt") {
    const { chunk, fek, fileId, chunkIndex } = e.data;
    try {
      const iv = new ArrayBuffer(12);
      new DataView(iv).setUint32(0, chunkIndex, false);
      const aadEnc = new TextEncoder().encode(fileId);
      const aadIdx = new ArrayBuffer(4);
      new DataView(aadIdx).setUint32(0, chunkIndex, false);
      const aad = new Uint8Array(aadEnc.length + 4);
      aad.set(aadEnc, 0);
      aad.set(new Uint8Array(aadIdx), aadEnc.length);

      const key = await crypto.subtle.importKey(
        "raw", fek, { name: "AES-GCM", length: 256 }, false, ["encrypt"]
      );
      const ciphertext = await crypto.subtle.encrypt(
        { name: "AES-GCM", iv, additionalData: aad.buffer }, key, chunk
      );
      const result = new Uint8Array(12 + ciphertext.byteLength);
      result.set(new Uint8Array(iv), 0);
      result.set(new Uint8Array(ciphertext), 12);
      ctx.postMessage({ type: "encrypted", encrypted: result, chunkIndex }, [result.buffer]);
    } catch (err) {
      ctx.postMessage({ type: "error", message: String(err) });
    }
  }

  if (type === "decrypt") {
    const { encrypted, fek, fileId, chunkIndex } = e.data;
    try {
      const iv = encrypted.slice(0, 12);
      const ciphertext = encrypted.slice(12);
      const aadEnc = new TextEncoder().encode(fileId);
      const aadIdx = new ArrayBuffer(4);
      new DataView(aadIdx).setUint32(0, chunkIndex, false);
      const aad = new Uint8Array(aadEnc.length + 4);
      aad.set(aadEnc, 0);
      aad.set(new Uint8Array(aadIdx), aadEnc.length);

      const key = await crypto.subtle.importKey(
        "raw", fek, { name: "AES-GCM", length: 256 }, false, ["decrypt"]
      );
      const plain = await crypto.subtle.decrypt(
        { name: "AES-GCM", iv, additionalData: aad.buffer }, key, ciphertext
      );
      ctx.postMessage({ type: "decrypted", plaintext: plain }, [plain]);
    } catch (err) {
      ctx.postMessage({ type: "error", message: String(err) });
    }
  }
};