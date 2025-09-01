import Image from 'next/image';

export default function Topbar() {
  return (
    <header className="w-full h-16 bg-blue flex items-center justify-between px-6 border-b border-orange text-white shadow-md">
      <div className="flex items-center">
        <Image src="/mimir-aip-logo.svg" alt="Mimir AIP" width={32} height={32} />
        <span className="ml-2 font-bold text-lg">Mimir AIP</span>
      </div>
      <div className="flex items-center space-x-4">
        {/* User info, notifications, quick actions can go here */}
        <span className="text-orange font-semibold">Welcome!</span>
      </div>
    </header>
  );
}
