'use client';

import { useState } from "react";
import { StartupAnimation } from "./StartupAnimation";

export function AppShell({ children }: { children: React.ReactNode }) {
  const [showAnimation, setShowAnimation] = useState(true);

  return (
    <>
      {showAnimation && (
        <StartupAnimation onComplete={() => setShowAnimation(false)} />
      )}
      {children}
    </>
  );
}
