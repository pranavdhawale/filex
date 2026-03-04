"use client";

import { useState, useCallback, useEffect } from "react";
import DropZone from "@/components/upload/DropZone";
import TTLSelector from "@/components/upload/TTLSelector";
import EncryptionToggle from "@/components/upload/EncryptionToggle";
import UploadProgress from "@/components/upload/UploadProgress";
import CompletionCard from "@/components/upload/CompletionCard";
import { startUpload } from "@/lib/upload";
import type { TTLSeconds, EncryptionMode, UploadProgress as UploadProgressT, UploadResult } from "@/types";

const STORAGE_KEY = "filex_last_upload";

type Phase = "idle" | "uploading" | "done" | "error";

function loadSavedResult(): UploadResult | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    // Restore the Date object from its ISO string
    parsed.expiresAt = new Date(parsed.expiresAt);
    // If the link has already expired, discard it
    if (parsed.expiresAt < new Date()) {
      sessionStorage.removeItem(STORAGE_KEY);
      return null;
    }
    return parsed as UploadResult;
  } catch {
    return null;
  }
}

export default function HomePage() {
  const [file, setFile] = useState<File | null>(null);
  const [ttl, setTTL] = useState<TTLSeconds>(1800);
  const [mode, setMode] = useState<EncryptionMode>("anonymous");
  const [passphrase, setPassphrase] = useState("");
  const [phase, setPhase] = useState<Phase>("idle");
  const [progress, setProgress] = useState<UploadProgressT | null>(null);
  const [result, setResult] = useState<UploadResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Restore saved result on first mount
  useEffect(() => {
    const saved = loadSavedResult();
    if (saved) {
      setResult(saved);
      setPhase("done");
    }
  }, []);

  const reset = useCallback(() => {
    sessionStorage.removeItem(STORAGE_KEY);
    setFile(null);
    setPhase("idle");
    setProgress(null);
    setResult(null);
    setError(null);
  }, []);

  const handleUpload = async () => {
    if (!file) return;
    if (mode === "master" && !passphrase.trim()) {
      setError("A passphrase is required for master key mode.");
      return;
    }

    setPhase("uploading");
    setError(null);

    try {
      const res = await startUpload({
        file,
        ttlSeconds: ttl,
        encryptionMode: mode,
        passphrase: mode === "master" ? passphrase : undefined,
        onProgress: setProgress,
      });
      // Persist so page reloads retain the link
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify(res));
      setResult(res);
      setPhase("done");
    } catch (err) {
      setPhase("error");
      setError(err instanceof Error ? err.message : "Upload failed.");
    }
  };

  const uploading = phase === "uploading";
  const canUpload = !!file && !uploading;

  const isDone = phase === "done" && !!result;

  return (
    <main className="min-h-screen flex items-center justify-center p-4">
      <div
        className={`w-full flex gap-6 ${
          isDone ? "max-w-4xl flex-row items-start" : "max-w-lg flex-col"
        }`}
      >
        {/* Left column: always rendered */}
        <div className="flex-1 space-y-8 min-w-0">
          {/* Header */}
          <div className="space-y-1">
            <h1 className="text-xl font-semibold tracking-tight">FileX</h1>
            <p className="text-sm text-[#888]">
              End-to-end encrypted file sharing. Server sees nothing.
            </p>
          </div>

          {/* Upload Card */}
          <div className="border border-[#111] rounded-lg bg-[#050505] p-6 space-y-5">
            <DropZone onFileSelected={setFile} disabled={uploading || isDone} />

            <div className="border-t border-[#111]" />

            <TTLSelector value={ttl} onChange={setTTL} disabled={uploading || isDone} />
            <EncryptionToggle
              value={mode}
              onChange={setMode}
              passphrase={passphrase}
              onPassphraseChange={setPassphrase}
              disabled={uploading || isDone}
            />

            {error && (
              <p className="text-xs text-red-400/80 bg-red-500/5 border border-red-500/10 rounded px-3 py-2">
                {error}
              </p>
            )}

            <button
              onClick={handleUpload}
              disabled={!canUpload || isDone}
              className="
                w-full py-2.5 rounded-lg text-sm font-medium
                bg-white text-black
                hover:bg-white/90
                disabled:opacity-25 disabled:cursor-not-allowed
                transition-opacity duration-150
              "
            >
              {uploading ? "Uploading..." : "Encrypt & Upload"}
            </button>

            {phase === "uploading" && (
              <div className="pt-2 animate-fade-in">
                <UploadProgress
                  progress={
                    progress || {
                      phase: "encrypting",
                      chunksDone: 0,
                      totalChunks: 1,
                      bytesUploaded: 0,
                      totalBytes: file?.size || 0,
                      speedMBps: 0,
                      etaSecs: 0,
                    }
                  }
                />
              </div>
            )}
          </div>
        </div>

        {/* Right column: GPU-composited slide+fade — only opacity & transform animated */}
        <div
          className="flex-1 min-w-0"
          style={{
            opacity: isDone ? 1 : 0,
            transform: isDone ? "translateX(0)" : "translateX(20px)",
            transition: "opacity 350ms cubic-bezier(0.4,0,0.2,1), transform 350ms cubic-bezier(0.4,0,0.2,1)",
            pointerEvents: isDone ? "auto" : "none",
            willChange: "opacity, transform",
          }}
        >
          {result && (
            <div className="border border-[#111] rounded-lg bg-[#050505] p-6">
              <CompletionCard
                shareUrl={result.shareUrl}
                expiresAt={result.expiresAt}
                encryptionMode={mode}
                onReset={reset}
              />
            </div>
          )}
        </div>
      </div>
    </main>
  );
}
