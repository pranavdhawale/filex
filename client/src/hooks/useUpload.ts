import { useState, useCallback } from "react";
import { startUpload, type UploadProgress, type UploadResult } from "../lib/upload";

type Phase = "idle" | "uploading" | "done" | "error";

export function useUpload() {
  const [phase, setPhase] = useState<Phase>("idle");
  const [progress, setProgress] = useState<UploadProgress | null>(null);
  const [result, setResult] = useState<UploadResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const upload = useCallback(async (opts: {
    file: File;
    ttlSeconds: number;
    passphrase: string;
  }) => {
    setPhase("uploading");
    setError(null);
    try {
      const res = await startUpload({
        ...opts,
        onProgress: setProgress,
      });
      setResult(res);
      setPhase("done");
      return res;
    } catch (err) {
      setPhase("error");
      setError(err instanceof Error ? err.message : "Upload failed");
      throw err;
    }
  }, []);

  const reset = useCallback(() => {
    setPhase("idle");
    setProgress(null);
    setResult(null);
    setError(null);
  }, []);

  return { phase, progress, result, error, upload, reset };
}