"use client";

import type { UploadProgress as UploadProgressT } from "@/types";
import { Loader2 } from "lucide-react";

interface UploadProgressProps {
  progress: UploadProgressT;
}





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
          {progress.phase === "completing" ? "Finalizing" : "Encrypting & Uploading"}
        </span>
      </div>

      {/* Progress Bar */}
      <div className="space-y-1.5">
        <div className="flex justify-between text-xs text-[#888]">
          <span>{pct}%</span>
        </div>
        <div className="flex gap-0.5 h-1.5 w-full">
          {Array.from({ length: Math.max(1, progress.totalChunks) }).map((_, i) => (
            <div
              key={i}
              className={`flex-1 rounded-full transition-colors duration-300 ${
                i < progress.chunksDone ? "bg-blue-500" : "bg-[#222]"
              }`}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
