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
        <CheckCircle size={18} className="text-[var(--color-success)]" />
        <span className="text-sm font-medium text-[var(--text-main)]">Upload complete</span>
      </div>

      <div className="space-y-2">
        <label className="text-xs text-[var(--text-dim)]">Share link</label>
        <div className="flex gap-2">
          <input
            readOnly
            value={shareUrl}
            className="flex-1 bg-[var(--glass-item-bg)] border border-[var(--glass-border-dim)] rounded-lg px-3 py-2 text-sm text-[var(--text-main)] font-mono truncate"
          />
          <button
            onClick={copyUrl}
            className="px-3 py-2 rounded-lg bg-[var(--btn-bg-idle)] border border-[var(--glass-border-dim)] hover:bg-[var(--button-hover)] transition-colors"
          >
            <Copy size={14} className={copied ? "text-[var(--color-success)]" : "text-[var(--text-dim)]"} />
          </button>
        </div>
      </div>

      <div className="space-y-2">
        <label className="text-xs text-[var(--text-dim)]">Passphrase (share this separately)</label>
        <input
          readOnly
          value={passphrase}
          className="w-full bg-[var(--glass-item-bg)] border border-[var(--glass-border-dim)] rounded-lg px-3 py-2 text-sm text-[var(--text-main)] font-mono"
        />
      </div>

      <p className="text-xs text-[var(--text-dim)]">
        Expires {expiresAt.toLocaleDateString()} {expiresAt.toLocaleTimeString()}
      </p>

      <button
        onClick={onReset}
        className="w-full py-2.5 rounded-lg text-sm font-medium bg-[var(--btn-bg-idle)] border border-[var(--glass-border-dim)] hover:bg-[var(--button-hover)] transition-colors"
      >
        Upload another file
      </button>
    </div>
  );
}