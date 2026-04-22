import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useListPosts } from '@/api/generated/posts/posts'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Loader2, ChevronLeft, ChevronRight, FileText } from 'lucide-react'

export const Route = createFileRoute('/posts/')({
  component: PostsPage,
})

function getUserId(): number | null {
  const token = localStorage.getItem('token')
  if (!token) return null
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return Number(payload.sub)
  } catch {
    return null
  }
}

function PostsPage() {
  const userId = getUserId()
  const [cursor, setCursor] = useState<string | undefined>(undefined)
  const [history, setHistory] = useState<string[]>([])

  const { data: page, isLoading } = useListPosts(
    { userId: userId ?? 0, cursor, limit: 20 },
    { query: { enabled: userId !== null } },
  )

  const goNext = () => {
    if (page?.nextCursor) {
      setHistory((h) => [...h, cursor ?? ''])
      setCursor(page.nextCursor ?? undefined)
    }
  }

  const goPrev = () => {
    const prev = [...history]
    const last = prev.pop()
    setHistory(prev)
    setCursor(last || undefined)
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Posts</h1>
        <p className="text-muted-foreground mt-1">Your posts, newest first.</p>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div className="flex items-center gap-3">
            <CardTitle className="text-base">Posts</CardTitle>
            {userId !== null && (
              <Badge variant="secondary">user {userId}</Badge>
            )}
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="icon" onClick={goPrev} disabled={history.length === 0}>
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <Button variant="outline" size="icon" onClick={goNext} disabled={!page?.nextCursor}>
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {userId === null ? (
            <div className="flex items-center justify-center h-32 text-muted-foreground text-sm">
              Not authenticated.
            </div>
          ) : isLoading ? (
            <div className="flex items-center justify-center h-32 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin mr-2" /> Loading…
            </div>
          ) : !page?.data.length ? (
            <div className="flex flex-col items-center justify-center h-32 gap-2 text-muted-foreground text-sm">
              <FileText className="h-8 w-8 opacity-40" />
              No posts yet.
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Content</TableHead>
                  <TableHead className="w-44">Created</TableHead>
                  <TableHead className="w-44">Updated</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {page.data.map((post) => (
                  <TableRow key={post.id}>
                    <TableCell className="font-mono text-xs text-muted-foreground">{post.id}</TableCell>
                    <TableCell className="max-w-lg">
                      <p className="truncate">{post.content}</p>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {new Date(post.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {new Date(post.updatedAt).toLocaleString()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          {page?.nextCursor && (
            <div className="px-4 py-3 border-t text-xs text-muted-foreground">
              cursor: <span className="font-mono">{page.nextCursor}</span>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
