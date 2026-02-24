'use client';

import { useEffect, useState } from "react";
import styles from "./StartupAnimation.module.css";

interface StartupAnimationProps {
  onComplete: () => void;
}

export function StartupAnimation({ onComplete }: StartupAnimationProps) {
  const [step, setStep] = useState(0);

  useEffect(() => {
    // Sequence:
    // 0ms:   Initial — giant "B" glyph
    // 200ms: Morph — "Byte" slides in, "B" shrinks to match
    // 1000ms: Fade out begins
    // 1500ms: Complete → hand off to page
    const t1 = setTimeout(() => setStep(1), 200);
    const t2 = setTimeout(() => setStep(2), 1000);
    const t3 = setTimeout(() => onComplete(), 1500);
    return () => { clearTimeout(t1); clearTimeout(t2); clearTimeout(t3); };
  }, [onComplete]);

  return (
    <div className={`${styles.container} ${step >= 2 ? styles.fadeOut : ""}`}>
      <div className={styles.logoWrapper}>
        {/* "File" slides in from left */}
        <span className={`${styles.prefix} ${step >= 1 ? styles.prefixVisible : ""}`}>
          File
        </span>
        {/* "X" is the big anchor glyph that shrinks */}
        <span className={`${styles.suffix} ${step >= 1 ? styles.suffixShrunk : ""}`}>
          X
        </span>
      </div>
    </div>
  );
}
