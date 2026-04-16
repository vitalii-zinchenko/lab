import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import {
  useListApiKeys,
  useCreateApiKey,
  useRevokeApiKey,
  getListApiKeysQueryKey,
} from '@/api/generated/auth/auth'
import type { CreatedApiKey } from '@/api/model'
import { useQueryClient } from '@tanstack/react-query'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Trash2, Plus, Loader2, Copy, Check } from 'lucide-react'

export const Route = createFileRoute('/config/')({
  component: ConfigPage,
})

function getUserIdFromToken(): number | null {
  const token = localStorage.getItem('token')
  if (!token) return null
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return parseInt(payload.sub, 10)
  } catch {
    return null
  }
}

function getKeyStatus(key: { revoked_at?: string | null; expires_at?: string | null }): 'active' | 'revoked' | 'expired' {
  if (key.revoked_at) return 'revoked'
  if (key.expires_at && new Date(key.expires_at) < new Date()) return 'expired'
  return 'active'
}

function StatusBadge({ status }: { status: 'active' | 'revoked' | 'expired' }) {
  if (status === 'active') return <Badge variant="default">Active</Badge>
  if (status === 'revoked') return <Badge variant="destructive">Revoked</Badge>
  return <Badge variant="secondary">Expired</Badge>
}

function CreatedKeyBanner({ created, onDismiss }: { created: CreatedApiKey; onDismiss: () => void }) {
  const [copiedId, setCopiedId] = useState(false)
  const [copiedSecret, setCopiedSecret] = useState(false)

  const copy = (text: string, setter: (v: boolean) => void) => {
    navigator.clipboard.writeText(text)
    setter(true)
    setTimeout(() => setter(false), 2000)
  }

  return (
    <div className="rounded-md border border-green-200 bg-green-50 p-4 space-y-3">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-semibold text-green-800">API key created — save the secret now</p>
          <p className="text-xs text-green-700 mt-0.5">The client secret is shown only once and cannot be retrieved again.</p>
        </div>
        <button onClick={onDismiss} className="text-green-600 hover:text-green-800 text-xs underline ml-4 shrink-0">
          Dismiss
        </button>
      </div>
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <span className="text-xs text-green-700 w-24 shrink-0">Client ID</span>
          <code className="flex-1 bg-white border rounded px-2 py-1 text-xs font-mono truncate">
            {String(created.client_id)}
          </code>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 shrink-0"
            onClick={() => copy(String(created.client_id), setCopiedId)}
          >
            {copiedId ? <Check className="h-3 w-3 text-green-600" /> : <Copy className="h-3 w-3" />}
          </Button>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-green-700 w-24 shrink-0">Client Secret</span>
          <code className="flex-1 bg-white border rounded px-2 py-1 text-xs font-mono truncate">
            {created.client_secret}
          </code>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 shrink-0"
            onClick={() => copy(created.client_secret, setCopiedSecret)}
          >
            {copiedSecret ? <Check className="h-3 w-3 text-green-600" /> : <Copy className="h-3 w-3" />}
          </Button>
        </div>
      </div>
    </div>
  )
}

function ConfigPage() {
  const queryClient = useQueryClient()
  const { data: apiKeys = [], isLoading } = useListApiKeys()
  const createApiKey = useCreateApiKey({
    mutation: {
      onSuccess: (data) => {
        setCreatedKey(data)
        setName('')
        setExpiresAt('')
        queryClient.invalidateQueries({ queryKey: getListApiKeysQueryKey() })
      },
    },
  })
  const revokeApiKey = useRevokeApiKey({
    mutation: {
      onSuccess: () => queryClient.invalidateQueries({ queryKey: getListApiKeysQueryKey() }),
    },
  })

  const [name, setName] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [createdKey, setCreatedKey] = useState<CreatedApiKey | null>(null)

  const handleCreate = () => {
    const userId = getUserIdFromToken()
    if (!userId) return
    createApiKey.mutate({
      data: {
        user_id: userId,
        name: name.trim() || undefined,
        expires_at: expiresAt ? new Date(expiresAt).toISOString() : undefined,
      },
    })
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Config</h1>
        <p className="text-muted-foreground mt-1">Manage your API keys for programmatic access.</p>
      </div>

      {createdKey && (
        <CreatedKeyBanner created={createdKey} onDismiss={() => setCreatedKey(null)} />
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">New API Key</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-3 flex-wrap">
            <Input
              placeholder="Name (optional)"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
              className="max-w-xs"
            />
            <Input
              type="datetime-local"
              placeholder="Expires at (optional)"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
              className="max-w-xs"
            />
            <Button onClick={handleCreate} disabled={createApiKey.isPending}>
              {createApiKey.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
              Create
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">API Keys</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="flex items-center justify-center h-32 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin mr-2" /> Loading…
            </div>
          ) : apiKeys.length === 0 ? (
            <div className="flex items-center justify-center h-32 text-muted-foreground text-sm">
              No API keys yet. Create one above.
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Client ID</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Expires</TableHead>
                  <TableHead>Last used</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {apiKeys.map((key) => {
                  const status = getKeyStatus(key)
                  return (
                    <TableRow key={String(key.client_id)}>
                      <TableCell className="font-medium">{key.name ?? '—'}</TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {String(key.client_id).slice(0, 8)}…
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {new Date(key.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {key.expires_at ? new Date(key.expires_at).toLocaleString() : '—'}
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {key.last_used_at ? new Date(key.last_used_at).toLocaleString() : '—'}
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={status} />
                      </TableCell>
                      <TableCell>
                        {status === 'active' && (
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => revokeApiKey.mutate({ clientId: String(key.client_id) })}
                            disabled={revokeApiKey.isPending}
                            title="Revoke key"
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
