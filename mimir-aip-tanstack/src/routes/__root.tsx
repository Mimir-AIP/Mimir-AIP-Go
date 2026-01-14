import { createRootRoute, Link, Outlet } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'

export const Route = createRootRoute({
  component: () => (
    <>
      <div className="flex w-full min-h-screen">
        <div className="flex-1 flex flex-col">
          <header className="bg-blue/20 border-b border-orange/30 p-4">
            <div className="flex items-center justify-between">
              <h1 className="text-2xl font-bold text-orange">Mimir AIP - TanStack</h1>
              <nav className="flex gap-4">
                <Link 
                  to="/dashboard" 
                  className="text-white hover:text-orange transition-colors"
                  activeProps={{ className: 'text-orange' }}
                >
                  Dashboard
                </Link>
                <Link 
                  to="/performance" 
                  className="text-white hover:text-orange transition-colors"
                  activeProps={{ className: 'text-orange' }}
                >
                  Performance
                </Link>
              </nav>
            </div>
          </header>
          <main className="flex-1 p-6">
            <Outlet />
          </main>
        </div>
      </div>
      <TanStackRouterDevtools />
    </>
  ),
})
