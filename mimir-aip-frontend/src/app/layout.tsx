import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import Sidebar from "@/components/layout/Sidebar";
import Topbar from "@/components/layout/Topbar";
import { Toaster } from "@/components/ui/sonner";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

// Helper to get page title based on pathname
const getPageTitle = (path: string): string => {
  if (path.startsWith('/chat')) return "AI Agent Chat - Mimir AIP";
  if (path.startsWith('/dashboard')) return "Dashboard - Mimir AIP";
  if (path.startsWith('/pipelines')) return "Pipelines - Mimir AIP";
  if (path.startsWith('/ontologies')) return "Ontologies - Mimir AIP";
  if (path.startsWith('/models')) return "ML Models - Mimir AIP";
  if (path.startsWith('/digital-twins')) return "Digital Twins - Mimir AIP";
  if (path.startsWith('/workflows')) return "Workflow Orchestration - Mimir AIP";
  if (path.startsWith('/monitoring')) return "Monitoring - Mimir AIP";
  if (path.startsWith('/settings')) return "Settings - Mimir AIP";
  return "Mimir AIP - AI Pipeline Orchestration";
};

// Remove static metadata to allow dynamic titles
// export const metadata: Metadata = {
//   title: "Mimir AIP - AI Pipeline Orchestration", 
//   description: "Modern AI pipeline orchestration and automation platform",
// };

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const pathname = usePathname();
  
  useEffect(() => {
    document.title = getPageTitle(pathname);
  }, [pathname]);
  
  return (
    <html lang="en">
      <head>
        <title>{getPageTitle(pathname)}</title>
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased bg-navy text-white flex`}
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
