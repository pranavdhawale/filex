import { useState, useEffect } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { Download, AlertCircle } from "lucide-react";
import { useDownload } from "../../hooks/useDownload";
import { getShare } from "../../lib/api";

type Phase = "loading" | "prompt" | "downloading" | "done" | "error" | "expired";

export const Route = createFileRoute("/s/$shareSlug")({
  component: SharePage,
});

function SharePage() {
  const { shareSlug } = Route.useParams();
  const [phase, setPhase] = useState<Phase>("loading");
  const [passphrase, setPassphrase] = useState("");
  const [passphraseError, setPassphraseError] = useState<string | null>(null);
  const [files, setFiles] = useState<Array<{ slug: string; filename: string; size: number; contentType: string }>>([]);
  const [expiresAt, setExpiresAt] = useState<string>("");
  const { download } = useDownload();

  useEffect(() => {
    (async () => {
      try {
        const data = await getShare(shareSlug);
        setFiles(data.files);
        setExpiresAt(data.expires_at);
        setPhase("prompt");
      } catch {
        setPhase("expired");
      }
    })();
  }, [shareSlug]);

  const handleDownloadAll = async () => {
    if (!passphrase.trim()) {
      setPassphraseError("Passphrase is required.");
      return;
    }
    setPhase("downloading");
    setPassphraseError(null);
    for (const file of files) {
      try {
        await download({ slug: file.slug, passphrase });
      } catch {
        setPassphraseError("Incorrect passphrase.");
        setPhase("prompt");
        return;
      }
    }
    setPhase("done");
  };

  return (
    <main className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-lg space-y-8">
        <div className="space-y-1">
          <h1 className="text-xl font-semibold tracking-tight">FileX</h1>
          <p className="text-sm text-[var(--text-dim)]">Shared files</p>
        </div>

        <div className="border border-[var(--glass-border-dim)] rounded-lg bg-[var(--glass-bg)] backdrop-blur-xl shadow-[var(--glass-shadow)] p-6 flex flex-col gap-5">
          {phase === "loading" && <p className="text-sm text-[var(--text-dim)]">Loading...</p>}

          {(phase === "prompt" || phase === "downloading") && (
            <>
              <div className="space-y-2">
                <p className="text-sm text-[var(--text-main)] font-medium">{files.length} file{files.length !== 1 ? "s" : ""} shared with you</p>
                {files.map((f) => (
                  <div key={f.slug} className="text-xs text-[var(--text-dim)] font-mono truncate">{f.filename} ({(f.size / 1024 / 1024).toFixed(1)} MB)</div>
                ))}
              </div>

              <input
                type="password"
                placeholder="Enter passphrase..."
                value={passphrase}
                onChange={(e) => setPassphrase(e.target.value)}
                className="w-full bg-[var(--glass-item-bg)] border border-[var(--glass-border-dim)] rounded-lg px-4 py-2.5 text-sm text-[var(--text-main)] placeholder:text-[var(--text-dim)] hover:border-[var(--glass-border)] focus:border-[var(--glass-border)] transition-colors"
              />
              {passphraseError && <p className="text-xs text-red-400/80">{passphraseError}</p>}

              <button
                onClick={handleDownloadAll}
                disabled={phase === "downloading"}
                className="w-full flex items-center justify-center gap-2 py-2.5 rounded-lg text-sm font-medium !bg-white !text-black hover:!bg-white/90 disabled:opacity-50 transition-opacity"
              >
                <Download size={14} />
                {phase === "downloading" ? "Downloading..." : "Download All"}
              </button>
            </>
          )}

          {phase === "done" && <p className="text-sm text-[var(--color-success)]">All downloads complete!</p>}
          {phase === "expired" && (
            <div className="flex gap-3 items-start">
              <AlertCircle size={16} className="text-red-400/70 shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-[var(--text-main)]">Share not found</p>
                <p className="text-xs text-[var(--text-dim)] mt-1">This share has expired or never existed.</p>
              </div>
            </div>
          )}
        </div>

        <p className="text-center text-xs text-[var(--text-dim)]">Decryption happens entirely in your browser.</p>
      </div>
    </main>
  );
}