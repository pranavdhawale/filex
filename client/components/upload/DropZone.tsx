"use client";
import { useCallback, useState } from "react";
import { Upload, File as FileIcon, X } from "lucide-react";

interface DropZoneProps {
  onFileSelected: (file: File) => void;
  disabled?: boolean;
}

export default function DropZone({ onFileSelected, disabled }: DropZoneProps) {
  const [dragging, setDragging] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  const handleFile = useCallback(
    (file: File) => {
      setSelectedFile(file);
      onFileSelected(file);
    },
    [onFileSelected]
  );

  const onDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setDragging(false);
      if (disabled) return;
      const file = e.dataTransfer.files[0];
      if (file) handleFile(file);
    },
    [handleFile, disabled]
  );

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024)
      return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
    return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  };

  if (selectedFile) {
    return (
      <div className="flex items-center justify-between p-4 border border-[#1a1a1a] rounded-lg bg-[#0a0a0a] animate-fade-in">
        <div className="flex items-center gap-3 min-w-0">
          <div className="w-8 h-8 rounded flex items-center justify-center bg-[#111] shrink-0">
            <FileIcon size={14} className="text-[#888]" />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-medium truncate text-white/90">
              {selectedFile.name}
            </p>
            <p className="text-xs text-[#888]">{formatSize(selectedFile.size)}</p>
          </div>
        </div>
        {!disabled && (
          <button
            onClick={() => setSelectedFile(null)}
            className="ml-3 text-[#444] hover:text-[#888] transition-colors shrink-0"
          >
            <X size={14} />
          </button>
        )}
      </div>
    );
  }

  return (
    <label
      className={`
        relative flex flex-col items-center justify-center gap-3
        border border-dashed rounded-lg p-10 cursor-pointer
        transition-all duration-150
        ${dragging ? "border-blue-500/50 bg-blue-500/5" : "border-[#1a1a1a] hover:border-[#333]"}
        ${disabled ? "opacity-50 cursor-not-allowed" : ""}
      `}
      onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
      onDragLeave={() => setDragging(false)}
      onDrop={onDrop}
    >
      <input
        type="file"
        className="sr-only"
        disabled={disabled}
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) handleFile(file);
        }}
      />
      <div className="w-10 h-10 rounded-lg flex items-center justify-center bg-[#0f0f0f] border border-[#1a1a1a]">
        <Upload size={18} className="text-[#555]" />
      </div>
      <div className="text-center">
        <p className="text-sm text-white/70">
          <span className="text-blue-400 font-medium">Choose a file</span>
          {" or drag and drop"}
        </p>
        <p className="text-xs text-[#555] mt-1">Any file, up to 5 GB</p>
      </div>
    </label>
  );
}
