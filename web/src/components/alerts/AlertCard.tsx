import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { CrowdSecAlert, Decision } from '@/lib/api'
import { crowdsecAPI } from '@/lib/api/crowdsec'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { AlertTriangle, Info, Eye } from 'lucide-react'

interface AlertCardProps {
  alert: CrowdSecAlert
  index: number
  isExpanded: boolean
  onToggle?: () => void
  onOpenChange?: (open: boolean) => void
}

function AlertCard({ alert, isExpanded, onToggle, onOpenChange }: AlertCardProps) {
  const [inspectOpen, setInspectOpen] = useState(false)

  const handleOpenChange = (nextOpen: boolean) => {
    if (onOpenChange) {
      onOpenChange(nextOpen)
    } else if (onToggle) {
      onToggle()
    }
  }

  return (
    <>
    <Collapsible open={isExpanded} onOpenChange={handleOpenChange}>
      <Card className="border-l-4 border-l-orange-500 overflow-hidden">
        <CollapsibleTrigger asChild>
          <button
            type="button"
            aria-expanded={isExpanded}
            className="w-full text-left transition-colors hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          >
            <CardHeader className="p-3 sm:p-5 pb-3 sm:pb-3">
              <div className="flex items-center justify-between gap-2 min-w-0">
                <div className="flex items-center gap-2 sm:gap-3 min-w-0 flex-1">
                  <AlertTriangle className="h-5 w-5 text-orange-500 shrink-0 pointer-events-none" />
                  <div className="text-left min-w-0 flex-1">
                    <CardTitle
                      className="text-sm sm:text-base font-semibold truncate leading-tight"
                      title={alert.scenario}
                    >
                      {alert.scenario}
                    </CardTitle>
                    <CardDescription
                      className="text-xs sm:text-sm truncate mt-0.5"
                      title={`${alert.scope}: ${alert.value}`}
                    >
                      <span className="font-medium text-foreground/80">{alert.scope}:</span>{' '}
                      <span>{alert.value}</span>
                    </CardDescription>
                  </div>
                </div>
                <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
                  <Badge
                    variant={alert.type === 'ban' ? 'destructive' : 'default'}
                    className="text-[10px] sm:text-xs px-1.5 sm:px-2 py-0 sm:py-0.5 shrink-0"
                  >
                    {alert.type || 'Unknown'}
                  </Badge>
                  {alert.origin && (
                    <Badge
                      variant="secondary"
                      className="text-[10px] sm:text-xs px-1.5 sm:px-2 py-0 sm:py-0.5 hidden xs:inline-flex shrink-0"
                    >
                      {alert.origin}
                    </Badge>
                  )}
                  <div
                    className="p-1 rounded-md text-muted-foreground hover:text-foreground shrink-0"
                    title={isExpanded ? 'Collapse info' : 'Expand info'}
                  >
                    <Info className="h-4 w-4 pointer-events-none" />
                  </div>
                </div>
              </div>
            </CardHeader>
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <CardContent className="p-3 sm:p-5 pt-0 sm:pt-0">
            {/* Full Scenario Banner so the full name is always visible when clicking info */}
            <div className="rounded-md bg-muted/60 p-2.5 sm:p-3 mb-3 border">
              <div className="text-[11px] text-muted-foreground font-medium uppercase tracking-wider mb-1">
                Scenario
              </div>
              <div className="text-sm font-semibold break-all text-foreground select-all">
                {alert.scenario}
              </div>
            </div>

            <div className="grid gap-2 sm:gap-3 md:grid-cols-2 text-sm">
              <div>
                <span className="font-medium">Alert ID:</span>{' '}
                <span className="text-muted-foreground font-mono">#{alert.id}</span>
              </div>
              <div>
                <span className="font-medium">Target ({alert.scope}):</span>{' '}
                <span className="text-muted-foreground font-mono break-all">{alert.value}</span>
              </div>
              <div>
                <span className="font-medium">Origin:</span>{' '}
                <Badge variant="secondary" className="text-xs ml-1">{alert.origin}</Badge>
              </div>
              <div>
                <span className="font-medium">Events Count:</span>{' '}
                <span className="text-muted-foreground">{alert.events_count || 0}</span>
              </div>
              <div>
                <span className="font-medium">Start Time:</span>{' '}
                <span className="text-muted-foreground">
                  {new Date(alert.start_at).toLocaleString()}
                </span>
              </div>
              <div>
                <span className="font-medium">Stop Time:</span>{' '}
                <span className="text-muted-foreground">
                  {alert.stop_at ? new Date(alert.stop_at).toLocaleString() : 'Ongoing'}
                </span>
              </div>
              {alert.source?.cn && (
                <div>
                  <span className="font-medium">Country:</span>{' '}
                  <span className="text-muted-foreground">{alert.source.cn}</span>
                </div>
              )}
              {alert.source?.as_name && (
                <div className="md:col-span-2">
                  <span className="font-medium">AS Name:</span>{' '}
                  <span className="text-muted-foreground break-all">{alert.source.as_name}</span>
                </div>
              )}
              {alert.capacity && (
                <div>
                  <span className="font-medium">Capacity:</span>{' '}
                  <span className="text-muted-foreground">{alert.capacity}</span>
                </div>
              )}
              {alert.leakspeed && (
                <div>
                  <span className="font-medium">Leak Speed:</span>{' '}
                  <span className="text-muted-foreground">{alert.leakspeed}</span>
                </div>
              )}
              {alert.simulated !== undefined && (
                <div>
                  <span className="font-medium">Simulated:</span>{' '}
                  <Badge variant={alert.simulated ? 'outline' : 'default'} className="ml-1 text-xs">
                    {alert.simulated ? 'Yes' : 'No'}
                  </Badge>
                </div>
              )}
              {alert.message && (
                <div className="col-span-2">
                  <span className="font-medium">Message:</span>{' '}
                  <span className="text-muted-foreground break-words">{alert.message}</span>
                </div>
              )}
            </div>
            {alert.decisions && alert.decisions.length > 0 && (
              <div className="mt-4">
                <h4 className="font-medium mb-2">Associated Decisions</h4>
                <div className="rounded-md border overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Type</TableHead>
                        <TableHead>Value</TableHead>
                        <TableHead>Duration</TableHead>
                        <TableHead>Scope</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {alert.decisions.map((decision: Decision, idx: number) => (
                        <TableRow key={idx}>
                          <TableCell>
                            <Badge
                              variant={decision.type === 'ban' ? 'destructive' : 'default'}
                            >
                              {decision.type}
                            </Badge>
                          </TableCell>
                          <TableCell className="font-mono text-sm">
                            {decision.value}
                          </TableCell>
                          <TableCell>{decision.duration}</TableCell>
                          <TableCell>
                            <Badge variant="outline">{decision.scope}</Badge>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            )}
            {alert.id && (
              <div className="mt-4">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    setInspectOpen(true)
                  }}
                >
                  <Eye className="h-4 w-4 mr-2" />
                  Inspect Full Alert
                </Button>
              </div>
            )}
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
    {alert.id && (
      <AlertInspectDialog
        alertId={alert.id}
        open={inspectOpen}
        onOpenChange={setInspectOpen}
      />
    )}
    </>
  )
}

function AlertInspectDialog({ alertId, open, onOpenChange }: { alertId: number; open: boolean; onOpenChange: (open: boolean) => void }) {
  const { data, isLoading } = useQuery({
    queryKey: ['alert-inspect', alertId],
    queryFn: async () => {
      const response = await crowdsecAPI.inspectAlert(alertId)
      return response.data.data ?? null
    },
    enabled: open,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Alert #{alertId} Details</DialogTitle>
          <DialogDescription>Full inspection data from CrowdSec LAPI</DialogDescription>
        </DialogHeader>
        {isLoading ? (
          <div className="flex items-center justify-center py-8 text-muted-foreground">Loading...</div>
        ) : data ? (
          <pre className="p-4 bg-muted rounded-lg text-xs overflow-x-auto whitespace-pre-wrap font-mono">
            {JSON.stringify(data, null, 2)}
          </pre>
        ) : (
          <p className="text-muted-foreground text-sm">No data available</p>
        )}
      </DialogContent>
    </Dialog>
  )
}

export { AlertCard }
export type { AlertCardProps }
