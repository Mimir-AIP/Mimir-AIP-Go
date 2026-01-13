'use client';

import Link from 'next/link';
import Image from 'next/image';
import { useState } from 'react';
import { 
  ChevronDown, 
  ChevronRight, 
  LayoutDashboard,
  GitBranch,
  ListTodo,
  Brain,
  Network,
  Layers,
  Search,
  Copy,
  Workflow,
  MessageSquare,
  Activity,
  Puzzle,
  Settings,
  Sliders
} from 'lucide-react';
import { usePathname } from 'next/navigation';

interface NavItem {
  name: string;
  href?: string;
  icon?: React.ReactNode;
  children?: NavItem[];
}

const navItems: NavItem[] = [
  { name: 'Dashboard', href: '/dashboard', icon: <LayoutDashboard size={18} /> },
  { name: 'Pipelines', href: '/pipelines', icon: <GitBranch size={18} /> },
  { name: 'Jobs', href: '/jobs', icon: <ListTodo size={18} /> },
  { 
    name: 'Ontology & AI', 
    icon: <Brain size={18} />,
    children: [
      { name: 'Ontologies', href: '/ontologies', icon: <Network size={16} /> },
      { name: 'Knowledge Graph', href: '/knowledge-graph', icon: <Layers size={16} /> },
      { name: 'Entity Extraction', href: '/extraction', icon: <Search size={16} /> },
      { name: 'ML Models', href: '/models', icon: <Brain size={16} /> },
      { name: 'Digital Twins', href: '/digital-twins', icon: <Copy size={16} /> },
    ]
  },
  { name: 'Workflows', href: '/workflows', icon: <Workflow size={18} /> },
  { name: 'Agent Chat', href: '/chat', icon: <MessageSquare size={18} /> },
  { name: 'Monitoring', href: '/monitoring', icon: <Activity size={18} /> },
  { name: 'Plugins', href: '/plugins', icon: <Puzzle size={18} /> },
  { name: 'Config', href: '/config', icon: <Sliders size={18} /> },
  { name: 'Settings', href: '/settings', icon: <Settings size={18} /> },
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
          className={`w-full flex items-center justify-between px-4 py-2.5 rounded-lg font-medium transition-all ${
            isChildActive 
              ? 'bg-blue text-orange' 
              : 'hover:bg-blue/50 hover:text-orange'
          }`}
        >
          <span className="flex items-center gap-3">
            {item.icon && <span className="text-orange">{item.icon}</span>}
            {item.name}
          </span>
          {isOpen ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
        </button>
        {isOpen && item.children && (
          <ul className="mt-1 ml-6 space-y-1 border-l border-blue/50 pl-2">
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
        data-test={`sidebar-${item.href?.replace('/', '').replace('-', '')}-link`}
        className={`flex items-center gap-3 px-4 py-2.5 rounded-lg font-medium transition-all ${
          isActive
            ? 'bg-orange text-navy'
            : 'hover:bg-blue/50 hover:text-orange focus:bg-blue/50 focus:text-orange'
        }`}
      >
        {item.icon && <span className={isActive ? 'text-navy' : 'text-orange'}>{item.icon}</span>}
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
