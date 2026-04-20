import { useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { DropZone } from "../components/upload/DropZone";
import { TTLSelector } from "../components/upload/TTLSelector";
import { UploadProgress } from "../components/upload/UploadProgress";
import { CompletionCard } from "../components/upload/CompletionCard";
import { useUpload } from "../hooks/useUpload";

export const Route = createFileRoute("/")({
  component: HomePage,
});

function HomePage() {
  const [file, setFile] = useState<File | null>(null);
  const [ttl, setTtl] = useState<number>(1800);
  const [passphrase, setPassphrase] = useState("");
  const { phase, progress, result, error, upload, reset } = useUpload();

  const handleUpload = async () => {
    if (!file || !passphrase.trim()) return;
    await upload({ file, ttlSeconds: ttl, passphrase });
  };

  const isDone = phase === "done" && !!result;

  return (
    <main className="min-h-screen flex items-center justify-center p-4">
      <div className={`w-full flex gap-6 ${isDone ? "max-w-4xl flex-row items-start" : "max-w-lg flex-col"}`}>
        <div className="flex-1 space-y-8 min-w-0">
          <div className="space-y-1">
            <h1 className="text-xl font-semibold tracking-tight">FileX</h1>
            <p className="text-sm text-[#888]">End-to-end encrypted file sharing. Server sees nothing.</p>
          </div>

          <div className="border border-[#111] rounded-lg bg-[#050505] p-6 space-y-5">
            <DropZone onFileSelected={setFile} disabled={phase === "uploading"} />
            {file && <p className="text-sm text-white/70">{file.name} ({(file.size / 1024 / 1024).toFixed(1)} MB)</p>}
            <div className="border-t border-[#111]" />
            <TTLSelector value={ttl} onChange={setTtl} disabled={phase === "uploading"} />
            <div className="space-y-2">
              <label className="text-xs text-[#888]">Passphrase</label>
              <input
                type="password"
                placeholder="Enter a passphrase..."
                value={passphrase}
                onChange={(e) => setPassphrase(e.target.value)}
                disabled={phase === "uploading"}
                className="w-full bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg px-4 py-2.5 text-sm text-white/90 placeholder:text-[#444] hover:border-[#333] focus:border-[#444] transition-colors disabled:opacity-50"
              />
            </div>

            {error && (
              <p className="text-xs text-red-400/80 bg-red-500/5 border border-red-500/10 rounded px-3 py-2">
                {error}
              </p>
            )}

            <button
              onClick={handleUpload}
              disabled={!file || !passphrase.trim() || phase === "uploading"}
              className="w-full py-2.5 rounded-lg text-sm font-medium bg-white text-black hover:bg-white/90 disabled:opacity-25 disabled:cursor-not-allowed transition-opacity"
            >
              {phase === "uploading" ? "Encrypting & Uploading..." : "Encrypt & Upload"}
            </button>

            {phase === "uploading" && progress && (
              <div className="pt-2 animate-fade-in">
                <UploadProgress progress={progress} />
              </div>
            )}
          </div>
        </div>

        <div
          className="flex-1 min-w-0"
          style={{
            opacity: isDone ? 1 : 0,
            transform: isDone ? "translateX(0)" : "translateX(20px)",
            transition: "opacity 350ms cubic-bezier(0.4,0,0.2,1), transform 350ms cubic-bezier(0.4,0,0.2,1)",
            pointerEvents: isDone ? "auto" : "none",
          }}
        >
          {result && (
            <div className="border border-[#111] rounded-lg bg-[#050505] p-6">
              <CompletionCard
                shareUrl={result.shareUrl}
                passphrase={passphrase}
                expiresAt={result.expiresAt}
                onReset={() => { reset(); setFile(null); setPassphrase(""); }}
              />
            </div>
          )}
        </div>
      </div>
    </main>
  );
}