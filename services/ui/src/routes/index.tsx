import { createFileRoute } from '@tanstack/react-router'
import { useListItems } from '@/api/generated/items/items'
import { useGetUsageStats } from '@/api/generated/usage/usage'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Package, BarChart3, TrendingUp } from 'lucide-react'

export const Route = createFileRoute('/')({
  component: Dashboard,
})

const last30Days = {
  from: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
  to: new Date().toISOString(),
}

function Dashboard() {
  const { data: itemsData } = useListItems()
  const { data: statsData } = useGetUsageStats(last30Days)

  const totalUsage = statsData?.data.reduce((sum, d) => sum + d.count, 0) ?? 0
  const todayUsage = statsData?.data.at(-1)?.count ?? 0

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground mt-1">Overview of your Lab API.</p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Items</CardTitle>
            <Package className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{itemsData?.length ?? '—'}</div>
            <p className="text-xs text-muted-foreground mt-1">Stored items in the API</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Usage (all time)</CardTitle>
            <BarChart3 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalUsage.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground mt-1">API calls recorded</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Usage Today</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{todayUsage.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground mt-1">Calls on the most recent day</p>
          </CardContent>
        </Card>
      </div>

      {statsData && statsData.data.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Daily Usage (last {statsData.data.length} days)</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {[...statsData.data].reverse().slice(0, 10).map((d) => (
                <div key={d.date} className="flex items-center gap-4">
                  <span className="w-28 text-sm text-muted-foreground">{d.date}</span>
                  <div className="flex-1 bg-secondary rounded-full h-2 overflow-hidden">
                    <div
                      className="bg-primary h-full rounded-full"
                      style={{ width: `${Math.round((d.count / (statsData.data[0]?.count || 1)) * 100)}%` }}
                    />
                  </div>
                  <span className="text-sm font-medium w-16 text-right">{d.count.toLocaleString()}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
