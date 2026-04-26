import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import "./StartupAnimation.css";

interface StartupAnimationProps {
  onComplete: () => void;
}

export function StartupAnimation({ onComplete }: StartupAnimationProps) {
  const [phase, setPhase] = useState<"reveal" | "typing" | "pausing" | "fading">("reveal");

  const typingText = "ile";
  const typingDuration = typingText.length * 0.06;
  const revealDuration = 0.3;
  const pauseDuration = 0.5;
  const totalDuration = revealDuration + typingDuration + pauseDuration + 0.5;

  useEffect(() => {
    const typingTimer = setTimeout(() => setPhase("typing"), revealDuration * 1000);
    const pauseTimer = setTimeout(() => setPhase("pausing"), (revealDuration + typingDuration) * 1000);
    const fadeTimer = setTimeout(() => setPhase("fading"), (revealDuration + typingDuration + pauseDuration) * 1000);
    const completeTimer = setTimeout(() => onComplete(), totalDuration * 1000);

    return () => {
      clearTimeout(typingTimer);
      clearTimeout(pauseTimer);
      clearTimeout(fadeTimer);
      clearTimeout(completeTimer);
    };
  }, [onComplete, revealDuration, typingDuration, pauseDuration, totalDuration]);

  return (
    <AnimatePresence>
      {phase !== "fading" && (
        <motion.div
          className="startup-container"
          initial={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.5, ease: "easeOut" }}
        >
          <div className="typewriter-wrapper">
            <div className="typewriter-container">
              <motion.span
                className="brand-letter"
                initial={{ opacity: 0, scale: 1.2 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ duration: revealDuration, ease: "easeOut" }}
              >
                F
              </motion.span>

              <motion.div
                className="typewriter-text-middle"
                initial={{ width: 0 }}
                animate={{ width: phase === "reveal" ? 0 : "auto" }}
                transition={{ duration: typingDuration, ease: "linear" }}
              >
                <span className="brand-letter">{typingText}</span>
              </motion.div>

              <motion.span
                className="brand-letter brand-letter-x"
                initial={{ opacity: 0, scale: 1.2 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ duration: revealDuration, ease: "easeOut" }}
              >
                X
              </motion.span>
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}