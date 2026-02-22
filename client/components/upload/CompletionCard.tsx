"use client";

import { useState } from "react";
import { Copy, Check, AlertTriangle, Lock } from "lucide-react";
import type { EncryptionMode } from "@/types";

interface CompletionCardProps {
  shareUrl: string;
  expiresAt: Date;
  encryptionMode: EncryptionMode;
  onReset: () => void;
}

export default function CompletionCard({
  shareUrl,
  expiresAt,
  encryptionMode,
  onReset,
}: CompletionCardProps) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    await navigator.clipboard.writeText(shareUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const expireStr = expiresAt.toLocaleDateString(undefined, {
    weekday: "short",
    month: "short",
    day: "numeric",
    year: "numeric",
  });

  return (
    <div className="space-y-5 animate-fade-in">
      {/* Header */}
      <div className="flex items-center gap-2">
        <div className="w-5 h-5 rounded-full bg-emerald-500/20 flex items-center justify-center">
          <Check size={10} className="text-emerald-400" />
        </div>
        <span className="text-sm font-medium text-white/90">File uploaded</span>
      </div>

      {/* Link Box */}
      <div className="space-y-1.5">
        <p className="text-xs text-[#888]">Share link</p>
        <div className="flex items-center gap-2 p-3 bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg">
          <Lock size={12} className="text-[#555] shrink-0" />
          <code className="flex-1 text-xs text-blue-400 truncate font-mono">
            {shareUrl}
          </code>
          <button
            onClick={copy}
            className="shrink-0 text-[#555] hover:text-white/70 transition-colors"
          >
            {copied ? (
              <Check size={14} className="text-emerald-400" />
            ) : (
              <Copy size={14} />
            )}
          </button>
        </div>
      </div>

      {/* Expiry */}
      <p className="text-xs text-[#888]">
        Expires{" "}
        <span className="text-white/60">{expireStr}</span>
      </p>

      {/* Warning */}
      <div className="flex gap-2 p-3 bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg">
        <AlertTriangle
          size={12}
          className="shrink-0 mt-0.5 text-yellow-500/60"
        />
        <p className="text-xs text-[#888]">
          {encryptionMode === "anonymous"
            ? "This link contains the decryption key. If lost, the file is permanently unrecoverable. No one, including the server, can decrypt it."
            : "Your passphrase is required to decrypt this file. Keep it safe — it is never sent to the server."}
        </p>
      </div>

      {/* Reset */}
      <button
        onClick={onReset}
        className="text-xs text-[#555] hover:text-[#888] transition-colors underline underline-offset-2"
      >
        Upload another file
      </button>
    </div>
  );
}
