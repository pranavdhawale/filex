"use client";

import { ShieldCheck, Key, Info } from "lucide-react";
import type { EncryptionMode } from "@/types";

interface EncryptionToggleProps {
  value: EncryptionMode;
  onChange: (v: EncryptionMode) => void;
  passphrase: string;
  onPassphraseChange: (v: string) => void;
  disabled?: boolean;
}

export default function EncryptionToggle({
  value,
  onChange,
  passphrase,
  onPassphraseChange,
  disabled,
}: EncryptionToggleProps) {
  return (
    <div className="space-y-3">
      <label className="text-xs text-[#888] tracking-wide uppercase">
        Encryption mode
      </label>
      <div className="grid grid-cols-2 gap-2">
        {(["anonymous", "master"] as EncryptionMode[]).map((mode) => (
          <button
            key={mode}
            type="button"
            disabled={disabled}
            onClick={() => onChange(mode)}
            className={`
              flex items-center gap-2 px-3 py-2.5 rounded-lg border text-sm
              transition-all duration-150 disabled:opacity-50
              ${
                value === mode
                  ? "border-blue-500/50 bg-blue-500/10 text-white/90"
                  : "border-[#1a1a1a] text-[#888] hover:border-[#333] hover:text-white/70"
              }
            `}
          >
            {mode === "anonymous" ? (
              <ShieldCheck size={14} />
            ) : (
              <Key size={14} />
            )}
            <span className="capitalize">{mode}</span>
          </button>
        ))}
      </div>

      {value === "anonymous" && (
        <div className="flex gap-2 p-3 bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg text-xs text-[#888]">
          <Info size={12} className="shrink-0 mt-0.5 text-blue-400/60" />
          <p>
            Your file is encrypted in the browser. The decryption key is embedded in
            the share link. If the link is lost, the file is{" "}
            <span className="text-white/60">permanently unrecoverable.</span>
          </p>
        </div>
      )}

      {value === "master" && (
        <div className="space-y-2">
          <div className="flex gap-2 p-3 bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg text-xs text-[#888]">
            <Info size={12} className="shrink-0 mt-0.5 text-emerald-400/60" />
            <p>
              Your file is encrypted with a passphrase you control. The server
              never sees your key. You must remember your passphrase to decrypt.
            </p>
          </div>
          <input
            type="password"
            placeholder="Enter passphrase..."
            value={passphrase}
            disabled={disabled}
            onChange={(e) => onPassphraseChange(e.target.value)}
            autoComplete="new-password"
            className="
              w-full bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg
              px-4 py-2.5 text-sm text-white/90 placeholder:text-[#444]
              hover:border-[#333] focus:border-[#444] transition-colors
              disabled:opacity-50
            "
          />
        </div>
      )}
    </div>
  );
}
