import { useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { Download, Loader2, AlertCircle, Lock } from "lucide-react";
import { useDownload } from "../../hooks/useDownload";
import { accessFile } from "../../lib/api";

type Phase = "loading" | "prompt" | "downloading" | "done" | "error" | "expired";

export const Route = createFileRoute("/f/$slug")({
  component: DownloadPage,
});

function DownloadPage() {
  const { slug } = Route.useParams();
  const [phase, setPhase] = useState<Phase>("loading");
  const [passphrase, setPassphrase] = useState("");
  const [passphraseError, setPassphraseError] = useState<string | null>(null);
  const { progress, download } = useDownload();

  const fetchAndDownload = async () => {
    if (!passphrase.trim()) {
      setPassphraseError("Passphrase is required.");
      return;
    }
    setPhase("downloading");
    setPassphraseError(null);
    try {
      await download({ slug, passphrase });
      setPhase("done");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "";
      if (msg === "PASSPHRASE_REQUIRED" || msg.includes("operation-specific") || msg.includes("decrypt")) {
        setPassphraseError("Incorrect passphrase.");
        setPhase("prompt");
      } else {
        setPhase("error");
      }
    }
  };

  const handleLoad = async () => {
    try {
      await accessFile(slug);
      setPhase("prompt");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "";
      if (msg === "FILE_NOT_FOUND" || msg === "DOWNLOAD_LIMIT") {
        setPhase("expired");
      } else {
        setPhase("error");
      }
    }
  };

  return (
    <main className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-lg space-y-8">
        <div className="space-y-1">
          <h1 className="text-xl font-semibold tracking-tight">FileX</h1>
          <p className="text-sm text-[#888]">Encrypted file access</p>
        </div>

        <div className="border border-[#111] rounded-lg bg-[#050505] p-6 space-y-5">
          {phase === "loading" && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <Lock size={14} className="text-[#555]" />
                <code className="text-sm text-blue-400 font-mono truncate">{decodeURIComponent(slug)}</code>
              </div>
              <button onClick={handleLoad} className="w-full py-2.5 rounded-lg text-sm font-medium bg-white text-black hover:bg-white/90 transition-opacity">
                Load file
              </button>
            </div>
          )}

          {phase === "prompt" && (
            <div className="space-y-4">
              <div>
                <p className="text-sm text-white/80 font-medium">Passphrase required</p>
                <p className="text-xs text-[#888] mt-0.5">Enter the passphrase to decrypt this file.</p>
              </div>
              <input
                type="password"
                placeholder="Enter passphrase..."
                value={passphrase}
                onChange={(e) => setPassphrase(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && fetchAndDownload()}
                autoFocus
                className="w-full bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg px-4 py-2.5 text-sm text-white/90 placeholder:text-[#444] hover:border-[#333] focus:border-[#444] transition-colors"
              />
              {passphraseError && <p className="text-xs text-red-400/80">{passphraseError}</p>}
              <button onClick={fetchAndDownload} className="w-full flex items-center justify-center gap-2 py-2.5 rounded-lg text-sm font-medium bg-white text-black hover:bg-white/90 transition-opacity">
                <Download size={14} />
                Decrypt & Download
              </button>
            </div>
          )}

          {phase === "downloading" && (
            <div className="space-y-4 animate-fade-in">
              <div className="flex items-center gap-2">
                <Loader2 size={14} className="animate-spin text-blue-400" />
                <span className="text-sm text-white/70">
                  {progress ? `${progress.phase}...` : "Starting..."}
                </span>
              </div>
              {progress && (
                <div className="h-0.5 bg-[#111] rounded-full overflow-hidden">
                  <div className="h-full bg-blue-500 rounded-full transition-all duration-300" style={{ width: `${progress.totalBytes > 0 ? Math.round((progress.receivedBytes / progress.totalBytes) * 100) : 0}%` }} />
                </div>
              )}
            </div>
          )}

          {phase === "done" && (
            <p className="text-sm text-green-400">Download complete!</p>
          )}

          {phase === "expired" && (
            <div className="flex gap-3 items-start">
              <AlertCircle size={16} className="text-red-400/70 shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-white/80">File not found</p>
                <p className="text-xs text-[#888] mt-1">This file has expired or never existed.</p>
              </div>
            </div>
          )}

          {phase === "error" && (
            <div className="flex gap-3 items-start">
              <AlertCircle size={16} className="text-red-400/70 shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-white/80">Something went wrong</p>
              </div>
            </div>
          )}
        </div>

        <p className="text-center text-xs text-[#333]">Decryption happens entirely in your browser. The server never has access to your file.</p>
      </div>
    </main>
  );
}