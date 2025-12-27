'use client';

import Image from 'next/image';
import Link from 'next/link';
import { Bell, HelpCircle, User } from 'lucide-react';

export default function Topbar() {
  return (
    <header className="w-full h-16 bg-gradient-to-r from-blue via-navy to-blue flex items-center justify-between px-6 border-b border-blue/50 text-white">
      {/* Mobile Logo (hidden on desktop) */}
      <div className="flex items-center md:hidden">
        <Image src="/mimir-aip-logo.svg" alt="Mimir AIP" width={32} height={32} />
        <span className="ml-2 font-bold text-lg text-orange">Mimir AIP</span>
      </div>
      
      {/* Breadcrumb / Page context (hidden on mobile) */}
      <div className="hidden md:flex items-center text-sm text-white/60">
        <span>Autonomous Intelligence Platform</span>
      </div>
      
      {/* Right side actions */}
      <div className="flex items-center gap-4">
        <Link 
          href="/chat" 
          className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-orange/10 border border-orange/30 text-orange hover:bg-orange/20 transition-colors text-sm"
        >
          <span>💬</span>
          <span>Ask Mimir</span>
        </Link>
        
        <button className="p-2 rounded-lg hover:bg-blue/50 transition-colors relative">
          <Bell size={20} className="text-white/70 hover:text-orange" />
          <span className="absolute top-1 right-1 w-2 h-2 bg-orange rounded-full"></span>
        </button>
        
        <button className="p-2 rounded-lg hover:bg-blue/50 transition-colors">
          <HelpCircle size={20} className="text-white/70 hover:text-orange" />
        </button>
        
        <div className="flex items-center gap-2 pl-4 border-l border-blue/50">
          <div className="w-8 h-8 rounded-full bg-orange/20 flex items-center justify-center">
            <User size={18} className="text-orange" />
          </div>
          <span className="hidden sm:inline text-sm text-white/80">Admin</span>
        </div>
      </div>
    </header>
  );
}
