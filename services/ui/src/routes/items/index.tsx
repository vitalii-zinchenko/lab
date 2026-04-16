import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useListItems, useCreateItem, useDeleteItem } from '@/api/generated/items/items'
import { useQueryClient } from '@tanstack/react-query'
import { getListItemsQueryKey } from '@/api/generated/items/items'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Trash2, Plus, Loader2 } from 'lucide-react'

export const Route = createFileRoute('/items/')({
  component: ItemsPage,
})

function ItemsPage() {
  const queryClient = useQueryClient()
  const { data: items = [], isLoading } = useListItems()
  const createItem = useCreateItem({
    mutation: {
      onSuccess: () => queryClient.invalidateQueries({ queryKey: getListItemsQueryKey() }),
    },
  })
  const deleteItem = useDeleteItem({
    mutation: {
      onSuccess: () => queryClient.invalidateQueries({ queryKey: getListItemsQueryKey() }),
    },
  })

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')

  const handleCreate = () => {
    if (!name.trim()) return
    createItem.mutate({ data: { name: name.trim(), description: description.trim() || undefined } })
    setName('')
    setDescription('')
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Items</h1>
        <p className="text-muted-foreground mt-1">Create and manage items via the API.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">New Item</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-3">
            <Input
              placeholder="Name *"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
              className="max-w-xs"
            />
            <Input
              placeholder="Description (optional)"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
              className="max-w-sm"
            />
            <Button onClick={handleCreate} disabled={!name.trim() || createItem.isPending}>
              {createItem.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
              Add
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="flex items-center justify-center h-32 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin mr-2" /> Loading…
            </div>
          ) : items.length === 0 ? (
            <div className="flex items-center justify-center h-32 text-muted-foreground text-sm">
              No items yet. Create one above.
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="font-mono text-xs text-muted-foreground">{item.id}</TableCell>
                    <TableCell className="font-medium">{item.name}</TableCell>
                    <TableCell className="text-muted-foreground">{item.description ?? '—'}</TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {new Date(item.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => deleteItem.mutate({ id: item.id })}
                        disabled={deleteItem.isPending}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
