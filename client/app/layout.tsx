import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Bytefile — Encrypted File Sharing",
  description:
    "End-to-end encrypted, anonymous file sharing. Server never sees your data.",
  metadataBase: new URL("https://bytefile.app"),
  openGraph: {
    title: "Bytefile — Encrypted File Sharing",
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
          href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600&display=swap"
          rel="stylesheet"
        />
      </head>
      <body className="min-h-screen bg-black text-white/90 font-sans antialiased">
        {children}
      </body>
    </html>
  );
}
