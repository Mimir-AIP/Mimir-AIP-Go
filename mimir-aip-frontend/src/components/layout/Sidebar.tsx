import Link from 'next/link';
import Image from 'next/image';

const navItems = [
  { name: 'Dashboard', href: '/dashboard' },
  { name: 'Data Ingestion', href: '/data/upload' },
  { name: 'Pipelines', href: '/pipelines' },
  { name: 'Jobs', href: '/jobs' },
  { name: 'Ontologies', href: '/ontologies' },
  { name: 'Knowledge Graph', href: '/knowledge-graph' },
  { name: 'Digital Twins', href: '/digital-twins' },
  { name: 'Extraction', href: '/extraction' },
  { name: 'Plugins', href: '/plugins' },
  { name: 'Config', href: '/config' },
  { name: 'Settings', href: '/settings' },
  { name: 'Auth', href: '/login' },
];

export default function Sidebar() {
  return (
    <aside className="h-screen w-64 bg-navy flex flex-col border-r border-blue text-white">
      <div className="flex items-center justify-center h-24 border-b border-blue">
        <Image src="/mimir-aip-logo.svg" alt="Mimir AIP" width={48} height={48} />
        <span className="ml-3 text-2xl font-bold text-orange">MIMIR AIP</span>
      </div>
      <nav className="flex-1 py-6">
        <ul className="space-y-2">
          {navItems.map(item => (
            <li key={item.name}>
              <Link href={item.href} className="block px-6 py-2 rounded-lg font-medium hover:bg-blue hover:text-orange transition-colors focus:bg-blue focus:text-orange active:bg-orange active:text-navy">
                {item.name}
              </Link>
            </li>
          ))}
        </ul>
      </nav>
    </aside>
  );
}
