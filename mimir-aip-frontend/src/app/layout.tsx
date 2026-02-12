import type { Metadata } from "next";
// import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import Sidebar from "@/components/layout/Sidebar";
import Topbar from "@/components/layout/Topbar";
import { Toaster } from "@/components/ui/sonner";

// const geistSans = Geist({
//   variable: "--font-geist-sans",
//   subsets: ["latin"],
// });

// const geistMono = Geist_Mono({
//   variable: "--font-geist-mono",
//   subsets: ["latin"],
// });

export const metadata: Metadata = {
  title: "Mimir AIP - AI Pipeline Orchestration",
  description: "Modern AI pipeline orchestration and automation platform",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className="antialiased bg-navy text-white flex"
      >
        <div className="flex w-full">
           {/* Sidebar */}
           <div className="hidden md:block">
             <Sidebar />
           </div>
           <div className="flex-1 flex flex-col min-h-screen">
             {/* Topbar */}
             <Topbar />
             <main className="flex-1 p-6">
               {children}
             </main>
           </div>
         </div>
         <Toaster />
      </body>
    </html>
  );
}
