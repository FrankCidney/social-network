import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Social Network",
  description: "Connect with friends and share your thoughts",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
