import { useState } from "react";
import { createShare } from "../../lib/api";

interface ShareCreationProps {
  fileIds: string[];
  onComplete: (shareSlug: string) => void;
}

export function ShareCreation({ fileIds, onComplete }: ShareCreationProps) {
  const [ttl, setTtl] = useState<1800 | 3600 | 86400>(86400);
  const [creating, setCreating] = useState(false);

  const handleCreate = async () => {
    setCreating(true);
    try {
      const result = await createShare({
        file_ids: fileIds,
        ttl_seconds: ttl,
        max_downloads: 0,
      });
      onComplete(result.shareSlug);
    } catch {
      // Handle error
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="space-y-4">
      <p className="text-sm text-white/80">
        Create a multi-file share link. All files will use the same passphrase.
      </p>
      <button
        onClick={handleCreate}
        disabled={creating || fileIds.length === 0}
        className="w-full py-2.5 rounded-lg text-sm font-medium bg-white text-black hover:bg-white/90 disabled:opacity-25"
      >
        {creating ? "Creating..." : "Create Share Link"}
      </button>
    </div>
  );
}