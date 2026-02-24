import type { Metadata } from "next";
import "./globals.css";
import Particles from "@/components/Particles";
import { AppShell } from "@/components/AppShell";

export const metadata: Metadata = {
  title: "FileX",
  description:
    "End-to-end encrypted, anonymous file sharing. Server never sees your data.",
  metadataBase: new URL("https://filex.app"),
  openGraph: {
    title: "FileX",
    description: "Share files with zero knowledge encryption.",
    type: "website",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="bg-black">
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link
          rel="preconnect"
          href="https://fonts.gstatic.com"
          crossOrigin="anonymous"
        />
        <link
          href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap"
          rel="stylesheet"
        />
      </head>
      <body className="min-h-screen bg-black text-white/90 font-sans antialiased">
        {/* Fixed full-screen particle background */}
        <div className="fixed inset-0 z-0 pointer-events-none">
          <Particles
            particleCount={300}
            particleSpread={10}
            speed={0.05}
            particleColors={["#ffffff", "#ffffff"]}
            moveParticlesOnHover={false}
            alphaParticles={false}
            particleBaseSize={120}
            sizeRandomness={1}
            cameraDistance={20}
            disableRotation={false}
            pixelRatio={1}
          />
        </div>
        {/* Startup animation + page content */}
        <AppShell>
          <div className="relative z-10">
            {children}
          </div>
        </AppShell>
        {/* Footer — matches notex glass-footer */}
        <p className="fixed bottom-[20px] left-1/2 -translate-x-1/2 z-10 text-[0.75rem] sm:text-[0.8rem] font-normal leading-normal text-white/65 pointer-events-none select-none whitespace-nowrap">
          Made with ❤️
        </p>
      </body>
    </html>
  );
}
