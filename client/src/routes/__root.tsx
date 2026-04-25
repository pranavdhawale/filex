import { useState } from "react";
import { createRootRoute, Outlet } from "@tanstack/react-router";
import { StartupAnimation } from "../components/StartupAnimation";
import Particles from "../components/Particles";

export const Route = createRootRoute({
  component: RootLayout,
});

function RootLayout() {
  const [showAnimation, setShowAnimation] = useState(true);

  return (
    <div className="min-h-screen bg-[var(--bg-gradient)] text-[var(--text-main)]">
      <Particles
        particleColors={["#ffffff", "#ffffff"]}
        particleCount={500}
        particleSpread={10}
        speed={0.1}
        particleBaseSize={120}
        moveParticlesOnHover={false}
        alphaParticles={false}
        disableRotation={true}
        pixelRatio={1}
        className="global-particles"
      />

      <div className="relative z-[1]">
        <Outlet />
      </div>

      {showAnimation && (
        <StartupAnimation onComplete={() => setShowAnimation(false)} />
      )}
    </div>
  );
}