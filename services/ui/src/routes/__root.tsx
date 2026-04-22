import { createRootRoute, Link, Outlet, useNavigate, useRouterState } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/router-devtools'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { LayoutDashboard, Package, Activity, BarChart3, LogOut, Settings, FileText } from 'lucide-react'
import { useEffect } from 'react'
import { cn } from '@/lib/utils'

const navItems = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/items', label: 'Items', icon: Package },
  { to: '/events', label: 'Events', icon: Activity },
  { to: '/usage', label: 'Usage', icon: BarChart3 },
  { to: '/posts', label: 'Posts', icon: FileText },
]

function AuthGuard({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate()
  const location = useRouterState({ select: (s) => s.location })
  const isLoginPage = location.pathname === '/login'

  useEffect(() => {
    if (!isLoginPage && !localStorage.getItem('token')) {
      navigate({ to: '/login' })
    }
  }, [isLoginPage, navigate])

  return <>{children}</>
}

function RootLayout() {
  const navigate = useNavigate()
  const location = useRouterState({ select: (s) => s.location })
  const isLoginPage = location.pathname === '/login'

  if (isLoginPage) {
    return <Outlet />
  }

  const handleLogout = () => {
    localStorage.removeItem('token')
    navigate({ to: '/login' })
  }

  return (
    <div className="flex min-h-screen bg-background">
      <aside className="w-56 shrink-0 border-r bg-card flex flex-col h-screen sticky top-0 overflow-hidden">
        <div className="px-6 py-5 border-b">
          <span className="text-lg font-semibold tracking-tight">Lab UI</span>
        </div>
        <nav className="flex-1 px-3 py-4 space-y-1">
          {navItems.map(({ to, label, icon: Icon }) => (
            <Link
              key={to}
              to={to}
              className={cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
              )}
              activeProps={{ className: 'bg-accent text-accent-foreground' }}
              activeOptions={to === '/' ? { exact: true } : undefined}
            >
              <Icon className="h-4 w-4" />
              {label}
            </Link>
          ))}
        </nav>
        <div className="px-3 py-4 border-t space-y-1">
          <Link
            to="/config"
            className={cn(
              'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
            )}
            activeProps={{ className: 'bg-accent text-accent-foreground' }}
          >
            <Settings className="h-4 w-4" />
            Config
          </Link>
          <button
            onClick={handleLogout}
            className="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
          >
            <LogOut className="h-4 w-4" />
            Sign out
          </button>
        </div>
      </aside>

      <main className="flex-1 overflow-auto p-8">
        <Outlet />
      </main>

      {import.meta.env.DEV && (
        <>
          <TanStackRouterDevtools position="bottom-right" />
          <ReactQueryDevtools initialIsOpen={false} />
        </>
      )}
    </div>
  )
}

export const Route = createRootRoute({
  component: () => (
    <AuthGuard>
      <RootLayout />
    </AuthGuard>
  ),
})
