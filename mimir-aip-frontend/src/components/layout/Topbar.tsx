'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Bell, HelpCircle, User, LogOut } from 'lucide-react';
import { useState } from 'react';

export default function Topbar() {
  const router = useRouter();
  const [showUserMenu, setShowUserMenu] = useState(false);

  const handleLogout = async () => {
    // Clear auth token cookie
    document.cookie = 'auth_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
    
    // Clear localStorage
    if (typeof window !== 'undefined') {
      localStorage.removeItem('auth_token');
    }
    
    // Call logout API if needed
    try {
      await fetch('/api/v1/auth/logout', { method: 'POST' });
    } catch (error) {
      console.error('Logout error:', error);
    }
    
    // Redirect to login
    router.push('/login');
  };

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
        
        <div className="relative pl-4 border-l border-blue/50">
          <button
            onClick={() => setShowUserMenu(!showUserMenu)}
            className="flex items-center gap-2 hover:opacity-80 transition-opacity"
            data-testid="user-menu"
            aria-label="User menu"
          >
            <div className="w-8 h-8 rounded-full bg-orange/20 flex items-center justify-center">
              <User size={18} className="text-orange" />
            </div>
            <span className="hidden sm:inline text-sm text-white/80">Admin</span>
          </button>
          
          {/* User dropdown menu */}
          {showUserMenu && (
            <div className="absolute right-0 top-full mt-2 w-48 bg-navy border border-blue/50 rounded-lg shadow-xl z-50">
              <div className="p-2">
                <div className="px-3 py-2 text-sm text-white/60 border-b border-blue/30">
                  <div className="font-medium text-white">Admin</div>
                  <div className="text-xs">admin@mimir-aip.com</div>
                </div>
                <Link
                  href="/settings"
                  className="flex items-center gap-2 px-3 py-2 text-sm text-white/80 hover:bg-blue/30 rounded mt-1"
                  onClick={() => setShowUserMenu(false)}
                >
                  <User size={16} />
                  <span>Profile</span>
                </Link>
                <button
                  onClick={handleLogout}
                  className="flex items-center gap-2 w-full px-3 py-2 text-sm text-white/80 hover:bg-blue/30 rounded"
                >
                  <LogOut size={16} />
                  <span>Logout</span>
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
