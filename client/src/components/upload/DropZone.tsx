import { useCallback, useState } from "react";

interface DropZoneProps {
  onFileSelected: (file: File) => void;
  disabled?: boolean;
}

export function DropZone({ onFileSelected, disabled }: DropZoneProps) {
  const [dragOver, setDragOver] = useState(false);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setDragOver(false);
      if (disabled) return;
      const file = e.dataTransfer.files[0];
      if (file) onFileSelected(file);
    },
    [onFileSelected, disabled]
  );

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) onFileSelected(file);
    },
    [onFileSelected]
  );

  return (
    <div
      onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
      onDragLeave={() => setDragOver(false)}
      onDrop={handleDrop}
      className={`
        border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors
        ${dragOver ? "border-[var(--color-primary)] bg-[var(--color-primary)]/5" : "border-[var(--glass-border)] hover:border-[var(--glass-border)]"}
        ${disabled ? "opacity-50 cursor-not-allowed" : ""}
      `}
      onClick={() => !disabled && document.getElementById("file-input")?.click()}
    >
      <input
        id="file-input"
        type="file"
        className="hidden"
        onChange={handleChange}
        disabled={disabled}
      />
      <p className="text-sm text-[var(--text-dim)]">
        {disabled ? "Uploading..." : "Drop a file here or click to browse"}
      </p>
    </div>
  );
}