import { CheckCircle, Copy } from "lucide-react";
import { useState } from "react";

interface CompletionCardProps {
  shareUrl: string;
  passphrase: string;
  expiresAt: Date;
  onReset: () => void;
}

export function CompletionCard({ shareUrl, passphrase, expiresAt, onReset }: CompletionCardProps) {
  const [copied, setCopied] = useState(false);

  const copyUrl = () => {
    navigator.clipboard.writeText(shareUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <CheckCircle size={18} className="text-green-400" />
        <span className="text-sm font-medium text-white/80">Upload complete</span>
      </div>

      <div className="space-y-2">
        <label className="text-xs text-[#888]">Share link</label>
        <div className="flex gap-2">
          <input
            readOnly
            value={shareUrl}
            className="flex-1 bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg px-3 py-2 text-sm text-white/90 font-mono truncate"
          />
          <button
            onClick={copyUrl}
            className="px-3 py-2 rounded-lg bg-[#111] border border-[#1a1a1a] hover:bg-[#1a1a1a] transition-colors"
          >
            <Copy size={14} className={copied ? "text-green-400" : "text-[#888]"} />
          </button>
        </div>
      </div>

      <div className="space-y-2">
        <label className="text-xs text-[#888]">Passphrase (share this separately)</label>
        <input
          readOnly
          value={passphrase}
          className="w-full bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg px-3 py-2 text-sm text-white/90 font-mono"
        />
      </div>

      <p className="text-xs text-[#555]">
        Expires {expiresAt.toLocaleDateString()} {expiresAt.toLocaleTimeString()}
      </p>

      <button
        onClick={onReset}
        className="w-full py-2.5 rounded-lg text-sm font-medium bg-[#111] border border-[#1a1a1a] hover:bg-[#1a1a1a] transition-colors"
      >
        Upload another file
      </button>
    </div>
  );
}