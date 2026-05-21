import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "HookDrop — Drop Inspector",
  description: "Inspect incoming webhooks in real time",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full">
      <body className="h-full">{children}</body>
    </html>
  );
}
