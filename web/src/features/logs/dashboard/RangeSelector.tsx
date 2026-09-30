import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { AlertTriangle } from 'lucide-react'
import { DASHBOARD_RANGES, type DashboardRange } from '@/lib/api/dashboard'
import { cn } from '@/lib/utils'

/** Duration in milliseconds for each preset range. */
export const RANGE_DURATION_MS: Record<DashboardRange, number | null> = {
  '5m':  5 * 60_000,
  '1h':  60 * 60_000,
  '6h':  6 * 60 * 60_000,
  '24h': 24 * 60 * 60_000,
  '7d':  7 * 24 * 60 * 60_000,
  'all': null,
}

interface RangeSelectorProps {
  value: DashboardRange
  onChange: (next: DashboardRange) => void
  /** ISO-8601 timestamp of the oldest log entry. When provided, ranges
   *  older than the data are hidden so the user doesn't see empty graphs. */
  oldestEntry?: string
  className?: string
}

export function RangeSelector({ value, onChange, className }: RangeSelectorProps) {
  return (
    <TooltipProvider delayDuration={300}>
      <div className={cn('inline-flex items-center gap-1 rounded-md bg-muted p-1', className)}>
        {DASHBOARD_RANGES.map((r) => {
          const isAll = r === 'all'
          const isActive = value === r

          const btn = (
            <Button
              key={r}
              size="sm"
              variant={isActive ? 'default' : 'ghost'}
              onClick={() => onChange(r)}
              className={cn(
                'h-7 px-3 text-xs gap-1.5',
                isAll && !isActive && 'text-amber-500 hover:text-amber-600',
                isAll && isActive && 'bg-amber-500 text-white hover:bg-amber-600',
              )}
            >
              {isAll && (
                <AlertTriangle className="h-3 w-3 shrink-0" />
              )}
              {r}
            </Button>
          )

          if (isAll) {
            return (
              <Tooltip key={r}>
                <TooltipTrigger asChild>{btn}</TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-[220px] text-center text-xs">
                  <p className="font-semibold mb-1">⚠ Heavy Query</p>
                  <p className="text-muted-foreground">
                    Scans up to 500 000 log lines. May be slow on large deployments.
                  </p>
                </TooltipContent>
              </Tooltip>
            )
          }

          return btn
        })}
      </div>
    </TooltipProvider>
  )
}
