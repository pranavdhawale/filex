import { useState, useCallback } from "react";
import { startDownload, type DownloadProgress } from "../lib/download";

type Phase = "idle" | "downloading" | "done" | "error";

export function useDownload() {
  const [phase, setPhase] = useState<Phase>("idle");
  const [progress, setProgress] = useState<DownloadProgress | null>(null);
  const [error, setError] = useState<string | null>(null);

  const download = useCallback(async (opts: {
    slug: string;
    passphrase: string;
  }) => {
    setPhase("downloading");
    setError(null);
    try {
      await startDownload({
        ...opts,
        onProgress: setProgress,
      });
      setPhase("done");
    } catch (err) {
      setPhase("error");
      setError(err instanceof Error ? err.message : "Download failed");
      throw err;
    }
  }, []);

  const reset = useCallback(() => {
    setPhase("idle");
    setProgress(null);
    setError(null);
  }, []);

  return { phase, progress, error, download, reset };
}