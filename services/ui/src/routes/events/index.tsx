import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useCreateEvent, useCreateChEvent } from '@/api/generated/events/events'
import { EventLevel } from '@/api/model/eventLevel'
import type { NewEvent } from '@/api/model'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Loader2, Send } from 'lucide-react'

export const Route = createFileRoute('/events/')({
  component: EventsPage,
})

const LEVEL_VARIANT: Record<EventLevel, 'default' | 'destructive' | 'secondary' | 'outline'> = {
  error: 'destructive',
  warn: 'outline',
  info: 'secondary',
}

function EventForm({ title, description, onSubmit, isPending }: {
  title: string
  description: string
  onSubmit: (e: NewEvent) => void
  isPending: boolean
}) {
  const [eventType, setEventType] = useState('')
  const [details, setDetails] = useState('')
  const [level, setLevel] = useState<EventLevel>(EventLevel.info)

  const handleSend = () => {
    if (!eventType.trim()) return
    onSubmit({ event_type: eventType.trim(), details: details.trim() || undefined, level })
    setEventType('')
    setDetails('')
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-3 flex-wrap">
          <Input
            placeholder="event_type *"
            value={eventType}
            onChange={(e) => setEventType(e.target.value)}
            className="max-w-xs"
          />
          <Input
            placeholder="Details (optional)"
            value={details}
            onChange={(e) => setDetails(e.target.value)}
            className="max-w-sm"
          />
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">Level:</span>
          {Object.values(EventLevel).map((l) => (
            <button
              key={l}
              onClick={() => setLevel(l)}
              className="focus:outline-none"
            >
              <Badge
                variant={level === l ? LEVEL_VARIANT[l] : 'outline'}
                className="cursor-pointer capitalize"
              >
                {l}
              </Badge>
            </button>
          ))}
          <div className="flex-1" />
          <Button onClick={handleSend} disabled={!eventType.trim() || isPending}>
            {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
            Send
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function EventsPage() {
  const [log, setLog] = useState<Array<{ ts: string; dest: string; type: string; level: EventLevel; ok: boolean }>>([])

  const pgEvent = useCreateEvent({
    mutation: {
      onSuccess: (_, vars) => setLog((l) => [{ ts: new Date().toISOString(), dest: 'postgres', type: vars.data.event_type, level: vars.data.level, ok: true }, ...l]),
      onError: (_, vars) => setLog((l) => [{ ts: new Date().toISOString(), dest: 'postgres', type: vars.data.event_type, level: vars.data.level, ok: false }, ...l]),
    },
  })

  const chEvent = useCreateChEvent({
    mutation: {
      onSuccess: (_, vars) => setLog((l) => [{ ts: new Date().toISOString(), dest: 'clickhouse', type: vars.data.event_type, level: vars.data.level, ok: true }, ...l]),
      onError: (_, vars) => setLog((l) => [{ ts: new Date().toISOString(), dest: 'clickhouse', type: vars.data.event_type, level: vars.data.level, ok: false }, ...l]),
    },
  })

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Events</h1>
        <p className="text-muted-foreground mt-1">Send events to PostgreSQL or ClickHouse backends.</p>
      </div>

      <EventForm
        title="PostgreSQL Event"
        description="Writes to the event_history table."
        onSubmit={(e) => pgEvent.mutate({ data: e })}
        isPending={pgEvent.isPending}
      />

      <EventForm
        title="ClickHouse Event"
        description="Writes to the ClickHouse ch_events table."
        onSubmit={(e) => chEvent.mutate({ data: e })}
        isPending={chEvent.isPending}
      />

      {log.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Activity Log</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 font-mono text-xs">
              {log.map((entry, i) => (
                <div key={i} className="flex items-center gap-3">
                  <span className="text-muted-foreground">{new Date(entry.ts).toLocaleTimeString()}</span>
                  <Badge variant={entry.ok ? 'secondary' : 'destructive'} className="text-xs">
                    {entry.ok ? 'OK' : 'ERR'}
                  </Badge>
                  <span className="text-muted-foreground">{entry.dest}</span>
                  <span>{entry.type}</span>
                  <Badge variant={LEVEL_VARIANT[entry.level]} className="text-xs capitalize">{entry.level}</Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
