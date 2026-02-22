"use client";

import type { UploadProgress as UploadProgressT } from "@/types";
import { Loader2 } from "lucide-react";

interface UploadProgressProps {
  progress: UploadProgressT;
}

function formatSpeed(mbps: number) {
  if (mbps < 1) return `${(mbps * 1024).toFixed(0)} KB/s`;
  return `${mbps.toFixed(1)} MB/s`;
}

function formatETA(secs: number) {
  if (secs < 60) return `${Math.ceil(secs)}s`;
  return `${Math.floor(secs / 60)}m ${Math.ceil(secs % 60)}s`;
}

const PHASE_LABEL = {
  encrypting: "Encrypting",
  uploading: "Uploading",
  completing: "Finalizing",
};

export default function UploadProgress({ progress }: UploadProgressProps) {
  const pct =
    progress.totalBytes > 0
      ? Math.round((progress.bytesUploaded / progress.totalBytes) * 100)
      : 0;

  return (
    <div className="space-y-5 animate-fade-in">
      {/* Phase */}
      <div className="flex items-center gap-2">
        <Loader2 size={14} className="animate-spin text-blue-400" />
        <span className="text-sm text-white/70">
          {PHASE_LABEL[progress.phase]}
          {progress.phase === "uploading" &&
            ` — ${progress.chunksDone}/${progress.totalChunks} chunks`}
        </span>
      </div>

      {/* Progress Bar */}
      <div className="space-y-1.5">
        <div className="flex justify-between text-xs text-[#888]">
          <span>{pct}%</span>
          {progress.phase === "uploading" && progress.speedMBps > 0 && (
            <span className="flex gap-3">
              <span>{formatSpeed(progress.speedMBps)}</span>
              <span>{formatETA(progress.etaSecs)} left</span>
            </span>
          )}
        </div>
        <div className="h-0.5 bg-[#111] rounded-full overflow-hidden">
          <div
            className="h-full bg-blue-500 rounded-full transition-all duration-300 ease-out"
            style={{ width: `${pct}%` }}
          />
        </div>
      </div>

      {/* Chunk status */}
      {progress.phase === "uploading" && (
        <div className="grid grid-cols-5 gap-1.5">
          {Array.from({ length: Math.min(progress.totalChunks, 20) }).map(
            (_, i) => (
              <div
                key={i}
                className={`h-0.5 rounded-full transition-colors duration-300 ${
                  i < progress.chunksDone ? "bg-blue-500" : "bg-[#1a1a1a]"
                }`}
              />
            )
          )}
        </div>
      )}
    </div>
  );
}
