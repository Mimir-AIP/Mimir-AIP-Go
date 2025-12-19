'use client';

import Link from 'next/link';
import Image from 'next/image';
import { useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { usePathname } from 'next/navigation';

interface NavItem {
  name: string;
  href?: string;
  children?: NavItem[];
}

const navItems: NavItem[] = [
  { name: 'Dashboard', href: '/dashboard' },
  { name: 'Data Ingestion', href: '/data/upload' },
  { name: 'Workflows', href: '/workflows' },
  { name: 'Pipelines', href: '/pipelines' },
  { name: 'Jobs', href: '/jobs' },
  { 
    name: 'Ontology System', 
    children: [
      { name: 'Browse Ontologies', href: '/ontologies' },
      { name: 'Upload Ontology', href: '/ontologies/upload' },
      { name: 'Knowledge Graph', href: '/knowledge-graph' },
      { name: 'Entity Extraction', href: '/extraction' },
      { name: 'ML Models', href: '/models' },
      { name: 'Digital Twins', href: '/digital-twins' },
    ]
  },
  { name: 'Agent Chat', href: '/chat' },
  { name: 'Monitoring', href: '/monitoring' },
  { name: 'Plugins', href: '/plugins' },
  { name: 'Config', href: '/config' },
  { name: 'Settings', href: '/settings' },
];

function NavItemComponent({ item, depth = 0 }: { item: NavItem; depth?: number }) {
  const [isOpen, setIsOpen] = useState(false);
  const pathname = usePathname();
  const hasChildren = item.children && item.children.length > 0;
  
  // Check if any child is active
  const isChildActive = hasChildren && item.children?.some(child => 
    child.href && pathname.startsWith(child.href)
  );
  
  const isActive = item.href ? pathname === item.href : isChildActive;

  if (hasChildren) {
    return (
      <li>
        <button
          onClick={() => setIsOpen(!isOpen)}
          className={`w-full flex items-center justify-between px-6 py-2 rounded-lg font-medium transition-colors ${
            isChildActive 
              ? 'bg-blue text-orange' 
              : 'hover:bg-blue hover:text-orange'
          }`}
        >
          <span>{item.name}</span>
          {isOpen ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
        </button>
        {isOpen && item.children && (
          <ul className="mt-1 ml-4 space-y-1">
            {item.children.map(child => (
              <NavItemComponent key={child.name} item={child} depth={depth + 1} />
            ))}
          </ul>
        )}
      </li>
    );
  }

  return (
    <li>
      <Link 
        href={item.href!} 
        className={`block px-6 py-2 rounded-lg font-medium transition-colors ${
          isActive
            ? 'bg-orange text-navy'
            : 'hover:bg-blue hover:text-orange focus:bg-blue focus:text-orange'
        }`}
      >
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
    </aside>
  );
}
