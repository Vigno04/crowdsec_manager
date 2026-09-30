import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import { AlertTriangle, Clock } from 'lucide-react'
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

export const RANGE_LABELS: Record<DashboardRange, string> = {
  '5m':  '5m (Last 5 mins)',
  '1h':  '1h (Last 1 hour)',
  '6h':  '6h (Last 6 hours)',
  '24h': '24h (Last 24 hours)',
  '7d':  '7d (Last 7 days)',
  'all': 'all (All time)',
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
    <div className={cn('inline-flex items-center', className)}>
      {/* Mobile view: compact dropdown select so it never clips off screen */}
      <div className="sm:hidden">
        <Select value={value} onValueChange={(next) => onChange(next as DashboardRange)}>
          <SelectTrigger className="h-9 w-auto min-w-[95px] text-xs gap-1.5 bg-muted/80 hover:bg-muted border border-border/40 font-medium px-2.5">
            <div className="flex items-center gap-1.5">
              {value === 'all' ? (
                <AlertTriangle className="h-3.5 w-3.5 text-amber-500 shrink-0" />
              ) : (
                <Clock className="h-3.5 w-3.5 opacity-70 shrink-0" />
              )}
              <span className={cn('font-semibold text-xs', value === 'all' && 'text-amber-500')}>{value}</span>
            </div>
          </SelectTrigger>
          <SelectContent align="end" className="text-xs min-w-[170px]">
            {DASHBOARD_RANGES.map((r) => {
              const isAll = r === 'all'
              return (
                <SelectItem key={r} value={r} className="text-xs">
                  {isAll ? (
                    <span className="flex items-center gap-1.5 text-amber-500 font-medium">
                      <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
                      all — All time (heavy)
                    </span>
                  ) : (
                    <span>{RANGE_LABELS[r] ?? r}</span>
                  )}
                </SelectItem>
              )
            })}
          </SelectContent>
        </Select>
      </div>

      {/* Desktop / tablet view: segmented button group with scroll protection */}
      <TooltipProvider delayDuration={300}>
        <div className="hidden sm:inline-flex items-center gap-1 rounded-md bg-muted p-1 max-w-full overflow-x-auto scrollbar-none">
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
                  'h-7 px-3 text-xs gap-1.5 shrink-0',
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
    </div>
  )
}
