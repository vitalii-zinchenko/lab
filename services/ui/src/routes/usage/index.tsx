import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useGetUsage, useGetUsageStats } from '@/api/generated/usage/usage'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Loader2, ChevronLeft, ChevronRight } from 'lucide-react'

export const Route = createFileRoute('/usage/')({
  component: UsagePage,
})

const last30Days = {
  from: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
  to: new Date().toISOString(),
}

function UsagePage() {
  const [cursor, setCursor] = useState<string | undefined>(undefined)
  const [history, setHistory] = useState<string[]>([])

  const { data: page, isLoading } = useGetUsage({ ...last30Days, cursor, limit: 20 })
  const { data: stats } = useGetUsageStats(last30Days)

  const goNext = () => {
    if (page?.next_cursor) {
      setHistory((h) => [...h, cursor ?? ''])
      setCursor(page.next_cursor ?? undefined)
    }
  }

  const goPrev = () => {
    const prev = [...history]
    const last = prev.pop()
    setHistory(prev)
    setCursor(last || undefined)
  }

  const totalUsage = stats?.data.reduce((s, d) => s + d.count, 0) ?? 0

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Usage</h1>
        <p className="text-muted-foreground mt-1">API usage records with cursor pagination.</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Total Recorded Calls</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalUsage.toLocaleString()}</div>
          </CardContent>
        </Card>

        {stats && stats.data.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-medium">Most Recent Day</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.data.at(-1)?.count.toLocaleString()}</div>
              <p className="text-xs text-muted-foreground mt-1">{stats.data.at(-1)?.date}</p>
            </CardContent>
          </Card>
        )}
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Usage Records</CardTitle>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="icon" onClick={goPrev} disabled={history.length === 0}>
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <Button variant="outline" size="icon" onClick={goNext} disabled={!page?.next_cursor}>
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="flex items-center justify-center h-32 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin mr-2" /> Loading…
            </div>
          ) : !page?.data.length ? (
            <div className="flex items-center justify-center h-32 text-muted-foreground text-sm">
              No usage records found.
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>User ID</TableHead>
                  <TableHead>Operation</TableHead>
                  <TableHead>Timestamp</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {page.data.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-mono text-xs text-muted-foreground">{row.id}</TableCell>
                    <TableCell>
                      <Badge variant="secondary">{row.user_id}</Badge>
                    </TableCell>
                    <TableCell className="font-medium">{row.operation}</TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {new Date(row.timestamp).toLocaleString()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          {page?.next_cursor && (
            <div className="px-4 py-3 border-t text-xs text-muted-foreground">
              cursor: <span className="font-mono">{page.next_cursor}</span>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
