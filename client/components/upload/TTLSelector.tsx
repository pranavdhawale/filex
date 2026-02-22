"use client";

import { ChevronDown } from "lucide-react";
import type { TTLDays } from "@/types";

interface TTLSelectorProps {
  value: TTLDays;
  onChange: (v: TTLDays) => void;
  disabled?: boolean;
}

const OPTIONS: { value: TTLDays; label: string }[] = [
  { value: 1, label: "1 Day" },
  { value: 7, label: "7 Days" },
  { value: 15, label: "15 Days" },
];

export default function TTLSelector({ value, onChange, disabled }: TTLSelectorProps) {
  return (
    <div className="space-y-1.5">
      <label className="text-xs text-[#888] tracking-wide uppercase">
        Expires after
      </label>
      <div className="relative">
        <select
          value={value}
          disabled={disabled}
          onChange={(e) => onChange(Number(e.target.value) as TTLDays)}
          className="
            w-full appearance-none bg-[#0a0a0a] border border-[#1a1a1a]
            rounded-lg px-4 py-2.5 text-sm text-white/90
            hover:border-[#333] focus:border-[#444]
            transition-colors cursor-pointer disabled:opacity-50
          "
        >
          {OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <ChevronDown
          size={14}
          className="absolute right-3 top-1/2 -translate-y-1/2 text-[#555] pointer-events-none"
        />
      </div>
    </div>
  );
}
