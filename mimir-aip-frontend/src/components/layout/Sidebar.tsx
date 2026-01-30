'use client';

import Link from 'next/link';
import Image from 'next/image';
import { useState } from 'react';
import { 
  LayoutDashboard,
  GitBranch,
  Network,
  Copy,
  MessageSquare,
  Brain
} from 'lucide-react';
import { usePathname } from 'next/navigation';

interface NavItem {
  name: string;
  href: string;
  icon: React.ReactNode;
}

const navItems: NavItem[] = [
  { name: 'Dashboard', href: '/dashboard', icon: <LayoutDashboard size={18} /> },
  { name: 'Pipelines', href: '/pipelines', icon: <GitBranch size={18} /> },
  { name: 'Ontologies', href: '/ontologies', icon: <Network size={18} /> },
  { name: 'Digital Twins', href: '/digital-twins', icon: <Copy size={18} /> },
  { name: 'ML Models', href: '/models', icon: <Brain size={18} /> },
  { name: 'Agent Chat', href: '/chat', icon: <MessageSquare size={18} /> },
];

function NavItemComponent({ item }: { item: NavItem }) {
  const pathname = usePathname();
  const isActive = pathname === item.href || pathname.startsWith(item.href);

  return (
    <li>
      <Link 
        href={item.href} 
        data-test={`sidebar-${item.href.replace('/', '').replace('-', '')}-link`}
        className={`flex items-center gap-3 px-4 py-2.5 rounded-lg font-medium transition-all ${
          isActive
            ? 'bg-orange text-navy'
            : 'hover:bg-blue/50 hover:text-orange focus:bg-blue/50 focus:text-orange'
        }`}
      >
        <span className={isActive ? 'text-navy' : 'text-orange'}>{item.icon}</span>
        {item.name}
      </Link>
    </li>
  );
}

export default function Sidebar() {
  return (
    <aside className="h-screen w-64 bg-navy flex flex-col border-r border-blue text-white overflow-y-auto">
      <div className="flex items-center justify-center h-24 border-b border-blue flex-shrink-0">
        <Image src="/mimir-aip-logo.svg" alt="Mimir AIP" width={48} height={48} />
        <span className="ml-3 text-2xl font-bold text-orange">MIMIR AIP</span>
      </div>
      <nav className="flex-1 py-6">
        <ul className="space-y-2">
          {navItems.map(item => (
            <NavItemComponent key={item.name} item={item} />
          ))}
        </ul>
      </nav>
      <div className="p-4 border-t border-blue">
        <p className="text-xs text-white/40 text-center">
          Mimir AIP - Autonomous Intelligence Platform
        </p>
      </div>
    </aside>
  );
}
