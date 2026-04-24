/// <reference types="vite/client" />

declare module "argon2-browser" {
  interface HashOptions {
    pass: string;
    salt: number[];
    type?: number;
    mem?: number;
    time?: number;
    parallel?: number;
    hashLen?: number;
  }
  interface HashResult {
    hash: ArrayBuffer;
    hashHex: string;
    encoded: string;
  }
  export function hash(options: HashOptions): Promise<HashResult>;
  export const ArgonType: { Argon2d: number; Argon2i: number; Argon2id: number };
}