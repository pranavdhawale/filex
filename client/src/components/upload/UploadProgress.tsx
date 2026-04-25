import type { UploadProgress as UploadProgressType } from "../../lib/upload";

interface UploadProgressProps {
  progress: UploadProgressType;
}

const phaseLabel: Record<string, string> = {
  encrypting: "Encrypting",
  uploading: "Uploading",
  completing: "Completing",
};

export function UploadProgress({ progress }: UploadProgressProps) {
  const pct = progress.totalBytes > 0
    ? Math.round((progress.bytesUploaded / progress.totalBytes) * 100)
    : 0;
  const speed = progress.speedMBps > 0 ? ` — ${progress.speedMBps.toFixed(1)} MB/s` : "";
  const eta = progress.etaSecs > 1 ? ` (${Math.ceil(progress.etaSecs)}s left)` : "";

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <span className="text-sm text-[var(--text-dim)]">
          {phaseLabel[progress.phase] || "Working"} — {pct}%{speed}{eta}
        </span>
      </div>
      <div className="h-0.5 bg-[var(--glass-border-dim)] rounded-full overflow-hidden">
        <div
          className="h-full bg-[var(--color-primary)] rounded-full transition-all duration-300"
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}