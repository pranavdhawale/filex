"use client";

import { useState, use } from "react";
import { Download, Loader2, AlertCircle, Lock } from "lucide-react";
import { startDownload } from "@/lib/download";
import { accessFile } from "@/lib/api";
import type { AccessFileResponse } from "@/types";

type Phase = "loading" | "prompt" | "ready" | "downloading" | "error" | "expired";

export default function DownloadPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [phase, setPhase] = useState<Phase>("loading");
  const [meta, setMeta] = useState<AccessFileResponse | null>(null);
  const [passphrase, setPassphrase] = useState("");
  const [passphraseError, setPassphraseError] = useState<string | null>(null);
  const [downloadProgress, setDownloadProgress] = useState<{ received: number; total: number } | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Fetch metadata on mount — we do it on user interaction to avoid bots hammering the API
  const fetchMeta = async () => {
    try {
      const data = await accessFile(id);
      setMeta(data);
      setPhase(data.encryption_mode === "master" ? "prompt" : "ready");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "";
      if (msg === "FILE_NOT_FOUND" || msg === "FILE_EXPIRED") {
        setPhase("expired");
      } else {
        setError(msg || "Could not retrieve file.");
        setPhase("error");
      }
    }
  };

  const download = async () => {
    if (!meta) return;

    if (meta.encryption_mode === "master" && !passphrase.trim()) {
      setPassphraseError("Passphrase is required.");
      return;
    }

    setPhase("downloading");
    setPassphraseError(null);

    try {
      await startDownload({
        fileId: id,
        passphrase: meta.encryption_mode === "master" ? passphrase : undefined,
        onProgress: (received, total) =>
          setDownloadProgress({ received, total }),
      });
      setPhase("ready");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "";
      if (msg === "PASSPHRASE_REQUIRED" || msg.includes("operation-specific")) {
        setPassphraseError("Incorrect passphrase.");
        setPhase("prompt");
      } else {
        setError(msg || "Decryption failed.");
        setPhase("error");
      }
    }
  };

  const pct =
    downloadProgress && downloadProgress.total > 0
      ? Math.round((downloadProgress.received / downloadProgress.total) * 100)
      : 0;

  return (
    <main className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-lg space-y-8">
        {/* Header */}
        <div className="space-y-1">
          <h1 className="text-xl font-semibold tracking-tight">FileX</h1>
          <p className="text-sm text-[#888]">Encrypted file access</p>
        </div>

        <div className="border border-[#111] rounded-lg bg-[#050505] p-6 space-y-5">
          {/* Initial state — show file ID and ask to fetch */}
          {phase === "loading" && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <Lock size={14} className="text-[#555]" />
                <code className="text-sm text-blue-400 font-mono truncate">
                  {id}
                </code>
              </div>
              <p className="text-sm text-[#888]">
                This file is client-side encrypted. Click below to load it.
              </p>
              <button
                onClick={fetchMeta}
                className="w-full py-2.5 rounded-lg text-sm font-medium bg-white text-black hover:bg-white/90 transition-opacity"
              >
                Load file
              </button>
            </div>
          )}

          {/* Master key prompt */}
          {(phase === "prompt") && (
            <div className="space-y-4">
              <div>
                <p className="text-sm text-white/80 font-medium">Passphrase required</p>
                <p className="text-xs text-[#888] mt-0.5">
                  This file was encrypted with a master passphrase. Enter it to decrypt.
                </p>
              </div>
              <input
                type="password"
                placeholder="Enter passphrase..."
                value={passphrase}
                onChange={(e) => setPassphrase(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && download()}
                autoFocus
                className="
                  w-full bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg
                  px-4 py-2.5 text-sm text-white/90 placeholder:text-[#444]
                  hover:border-[#333] focus:border-[#444] transition-colors
                "
              />
              {passphraseError && (
                <p className="text-xs text-red-400/80">{passphraseError}</p>
              )}
              <button
                onClick={download}
                className="w-full py-2.5 rounded-lg text-sm font-medium bg-white text-black hover:bg-white/90 transition-opacity"
              >
                Decrypt & Download
              </button>
            </div>
          )}

          {/* Anonymous — ready to download */}
          {phase === "ready" && (
            <div className="space-y-4">
              <div>
                <p className="text-sm text-white/80 font-medium">File ready</p>
                <p className="text-xs text-[#888] mt-0.5">
                  Decryption happens entirely in your browser.
                </p>
              </div>
              <button
                onClick={download}
                className="w-full flex items-center justify-center gap-2 py-2.5 rounded-lg text-sm font-medium bg-white text-black hover:bg-white/90 transition-opacity"
              >
                <Download size={14} />
                Decrypt & Download
              </button>
            </div>
          )}

          {/* Downloading */}
          {phase === "downloading" && (
            <div className="space-y-4 animate-fade-in">
              <div className="flex items-center gap-2">
                <Loader2 size={14} className="animate-spin text-blue-400" />
                <span className="text-sm text-white/70">
                  {downloadProgress ? `Decrypting — ${pct}%` : "Fetching encrypted file…"}
                </span>
              </div>
              {downloadProgress && (
                <div className="h-0.5 bg-[#111] rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-500 rounded-full transition-all duration-300"
                    style={{ width: `${pct}%` }}
                  />
                </div>
              )}
            </div>
          )}

          {/* Expired / not found */}
          {phase === "expired" && (
            <div className="flex gap-3 items-start">
              <AlertCircle size={16} className="text-red-400/70 shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-white/80">File not found</p>
                <p className="text-xs text-[#888] mt-1">
                  This file has expired or never existed. Encrypted files are permanently
                  deleted when their TTL expires.
                </p>
              </div>
            </div>
          )}

          {/* Generic error */}
          {phase === "error" && (
            <div className="flex gap-3 items-start">
              <AlertCircle size={16} className="text-red-400/70 shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-white/80">Something went wrong</p>
                {error && <p className="text-xs text-[#888] mt-1">{error}</p>}
              </div>
            </div>
          )}
        </div>

        <p className="text-center text-xs text-[#333]">
          Decryption happens entirely in your browser. The server never has access to your file.
        </p>
      </div>
    </main>
  );
}
