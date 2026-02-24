"use client";

import { useState, useCallback } from "react";
import DropZone from "@/components/upload/DropZone";
import TTLSelector from "@/components/upload/TTLSelector";
import EncryptionToggle from "@/components/upload/EncryptionToggle";
import UploadProgress from "@/components/upload/UploadProgress";
import CompletionCard from "@/components/upload/CompletionCard";
import { startUpload } from "@/lib/upload";
import type { TTLDays, EncryptionMode, UploadProgress as UploadProgressT, UploadResult } from "@/types";

type Phase = "idle" | "uploading" | "done" | "error";

export default function HomePage() {
  const [file, setFile] = useState<File | null>(null);
  const [ttl, setTTL] = useState<TTLDays>(1);
  const [mode, setMode] = useState<EncryptionMode>("anonymous");
  const [passphrase, setPassphrase] = useState("");
  const [phase, setPhase] = useState<Phase>("idle");
  const [progress, setProgress] = useState<UploadProgressT | null>(null);
  const [result, setResult] = useState<UploadResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const reset = useCallback(() => {
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
        ttlDays: ttl,
        encryptionMode: mode,
        passphrase: mode === "master" ? passphrase : undefined,
        onProgress: setProgress,
      });
      setResult(res);
      setPhase("done");
    } catch (err) {
      setPhase("error");
      setError(err instanceof Error ? err.message : "Upload failed.");
    }
  };

  const uploading = phase === "uploading";
  const canUpload = !!file && !uploading;

  return (
    <main className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-lg space-y-8">
        {/* Header */}
        <div className="space-y-1">
          <h1 className="text-xl font-semibold tracking-tight">FileX</h1>
          <p className="text-sm text-[#888]">
            End-to-end encrypted file sharing. Server sees nothing.
          </p>
        </div>

        {/* Card */}
        <div className="border border-[#111] rounded-lg bg-[#050505] p-6 space-y-5">
          {phase === "done" && result ? (
            <CompletionCard
              shareUrl={result.shareUrl}
              expiresAt={result.expiresAt}
              encryptionMode={mode}
              onReset={reset}
            />
          ) : phase === "uploading" && progress ? (
            <UploadProgress progress={progress} />
          ) : (
            <>
              <DropZone onFileSelected={setFile} disabled={uploading} />

              <div className="border-t border-[#111]" />

              <TTLSelector value={ttl} onChange={setTTL} disabled={uploading} />
              <EncryptionToggle
                value={mode}
                onChange={setMode}
                passphrase={passphrase}
                onPassphraseChange={setPassphrase}
                disabled={uploading}
              />

              {error && (
                <p className="text-xs text-red-400/80 bg-red-500/5 border border-red-500/10 rounded px-3 py-2">
                  {error}
                </p>
              )}

              <button
                onClick={handleUpload}
                disabled={!canUpload}
                className="
                  w-full py-2.5 rounded-lg text-sm font-medium
                  bg-white text-black
                  hover:bg-white/90
                  disabled:opacity-25 disabled:cursor-not-allowed
                  transition-opacity duration-150
                "
              >
                Encrypt & Upload
              </button>
            </>
          )}
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-[#333]">
          Files are encrypted in your browser before leaving your device.
        </p>
      </div>
    </main>
  );
}
