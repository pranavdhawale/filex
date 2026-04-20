interface TTLSelectorProps {
  value: number;
  onChange: (value: number) => void;
  disabled?: boolean;
}

const TTL_OPTIONS = [
  { label: "30 minutes", value: 1800 },
  { label: "1 hour", value: 3600 },
  { label: "1 day", value: 86400 },
];

export function TTLSelector({ value, onChange, disabled }: TTLSelectorProps) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(Number(e.target.value))}
      disabled={disabled}
      className="w-full bg-[#0a0a0a] border border-[#1a1a1a] rounded-lg px-4 py-2.5 text-sm text-white/90 hover:border-[#333] focus:border-[#444] transition-colors disabled:opacity-50"
    >
      {TTL_OPTIONS.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  );
}