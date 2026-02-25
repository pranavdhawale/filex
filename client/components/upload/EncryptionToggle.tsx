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
              relative flex items-center justify-between px-3 py-2.5 rounded-lg border text-sm
              transition-all duration-150 disabled:opacity-50
              ${
                value === mode
                  ? "border-blue-500/50 bg-blue-500/10 text-white/90"
                  : "border-[#1a1a1a] text-[#888] hover:border-[#333] hover:text-white/70"
              }
            `}
          >
            <div className="flex items-center gap-2">
              {mode === "anonymous" ? (
                <ShieldCheck size={14} />
              ) : (
                <Key size={14} />
              )}
              <span className="capitalize">{mode}</span>
            </div>

            <div className="group/tooltip flex items-center relative">
              <Info
                size={14}
                className={`transition-colors ${
                  value === mode
                    ? "text-blue-400/80 hover:text-blue-400"
                    : "text-[#666] hover:text-[#bbb]"
                }`}
              />
              <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 w-[220px] p-2.5 bg-[#111] border border-[#333] rounded-lg text-xs text-[#a3a3a3] opacity-0 invisible group-hover/tooltip:opacity-100 group-hover/tooltip:visible transition-all duration-200 z-[60] text-left shadow-xl pointer-events-none font-normal tracking-normal cursor-default">
                {mode === "anonymous" ? (
                  <>
                    Your file is encrypted in the browser. The decryption key is embedded in
                    the share link. If the link is lost, the file is{" "}
                    <span className="text-white/80">permanently unrecoverable.</span>
                  </>
                ) : (
                  <>
                    Your file is encrypted with a passphrase you control. The server
                    never sees your key. You must remember your passphrase to decrypt.
                  </>
                )}
                {/* Tooltip triangle */}
                <div className="absolute top-full left-1/2 -translate-x-1/2 border-[5px] border-transparent border-t-[#333]" />
                <div className="absolute top-[calc(100%-1px)] left-1/2 -translate-x-1/2 border-[5px] border-transparent border-t-[#111]" />
              </div>
            </div>
          </button>
        ))}
      </div>

      {value === "master" && (
        <div className="space-y-2 mt-2">
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
