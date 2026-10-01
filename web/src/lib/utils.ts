import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

export function formatDate(date: string | Date): string {
  return new Date(date).toLocaleString()
}

export function parseUserAgent(ua: string) {
  if (!ua) return { browser: 'Unknown', os: 'Unknown', cpu: 'Unknown', device: 'Unknown' }

  let browser = 'Unknown'
  let os = 'Unknown'
  let cpu = 'Unknown'
  let device = 'Unknown'

  const uaLower = ua.toLowerCase()

  // Browser detection
  if (ua.includes('Firefox/')) browser = 'Firefox ' + ua.split('Firefox/')[1].split(' ')[0]
  else if (ua.includes('Edg/')) browser = 'Edge ' + ua.split('Edg/')[1].split(' ')[0]
  else if (ua.includes('Chrome/')) browser = 'Chrome ' + ua.split('Chrome/')[1].split(' ')[0]
  else if (ua.includes('Safari/') && !ua.includes('Chrome')) browser = 'Safari ' + ua.split('Safari/')[1].split(' ')[0]
  else if (ua.includes('OPR/') || ua.includes('Opera/')) browser = 'Opera ' + (ua.split('OPR/')[1] || ua.split('Opera/')[1] || '').split(' ')[0]
  else if (ua === 'Chrome' || ua.startsWith('Chrome ')) browser = ua
  else if (ua === 'Firefox' || ua.startsWith('Firefox ')) browser = ua
  else if (ua === 'Edge' || ua.startsWith('Edge ')) browser = ua
  else if (ua === 'Safari' || ua.startsWith('Safari ')) browser = ua
  else if (ua === 'Opera' || ua.startsWith('Opera ')) browser = ua
  else if (uaLower.includes('curl')) browser = 'Curl'
  else if (uaLower.includes('postman')) browser = 'Postman'
  else if (uaLower.includes('axios')) browser = 'Axios'
  else if (ua.includes('Go-http-client')) browser = 'Go HTTP Client'
  else if (uaLower.includes('python')) browser = 'Python'
  else if (ua.includes('Googlebot')) browser = 'Googlebot'
  else if (ua.includes('bingbot')) browser = 'Bingbot'
  else if (ua.includes('DuckDuckBot')) browser = 'DuckDuckBot'
  else if (ua.includes('YandexBot') || ua.includes('Yandex')) browser = 'YandexBot'
  else {
    const firstWord = ua.split(/[\s/]/)[0]
    if (firstWord && firstWord.length <= 25) browser = firstWord
  }

  // OS detection
  if (ua.includes('Android')) {
    const ver = ua.match(/Android ([\d.]+)/)?.[1]
    os = ver ? `Android ${ver}` : 'Android'
  } else if (ua.includes('iPhone') || ua.includes('iPad')) {
    const ver = ua.match(/(?:OS|Version) ([\d_]+)/)?.[1]?.replace(/_/g, '.')
    os = ver ? `iOS ${ver}` : 'iOS'
    device = ua.includes('iPhone') ? 'iPhone' : 'iPad'
  } else if (ua.includes('Windows NT')) {
    const ver = ua.match(/Windows NT ([\d.]+)/)?.[1]
    os = ver === '10.0' ? 'Windows 10/11' : ver === '6.3' ? 'Windows 8.1' : ver === '6.2' ? 'Windows 8' : ver === '6.1' ? 'Windows 7' : ver === '6.0' ? 'Windows Vista' : ver === '5.1' ? 'Windows XP' : 'Windows'
  } else if (ua.includes('Windows')) {
    os = 'Windows'
  } else if (ua.includes('Mac OS X') || ua.includes('Macintosh')) {
    const ver = ua.match(/Mac OS X ([\d_]+)/)?.[1]?.replace(/_/g, '.')
    os = ver ? `macOS ${ver}` : 'macOS'
  } else if (ua.includes('CrOS')) {
    os = 'ChromeOS'
  } else if (ua.includes('Ubuntu')) {
    os = 'Ubuntu'
  } else if (ua.includes('Debian')) {
    os = 'Debian'
  } else if (ua.includes('Linux')) {
    os = 'Linux'
  }

  // CPU detection
  if (uaLower.includes('arm_64') || uaLower.includes('aarch64') || uaLower.includes('arm64')) cpu = 'ARM 64-bit'
  else if (uaLower.includes('x86_64') || uaLower.includes('amd64') || uaLower.includes('win64') || uaLower.includes('wow64') || uaLower.includes('x64') || ua.includes('Intel')) cpu = 'x86 64-bit'
  else if (uaLower.includes('i386') || uaLower.includes('i686') || uaLower.includes('x86')) cpu = 'x86 32-bit'
  else if (uaLower.includes('armv7') || uaLower.includes('armv8') || uaLower.includes('arm')) cpu = 'ARM'

  // Device detection
  if (ua.includes('iPhone')) {
    device = 'iPhone'
  } else if (ua.includes('iPad')) {
    device = 'iPad'
  } else if (ua.includes('Android')) {
    const deviceMatch = ua.match(/\(([^)]+)\)/)
    if (deviceMatch) {
      const parts = deviceMatch[1].split(';')
      for (const part of parts) {
        const p = part.trim()
        if (p.includes('Android') || p.includes('Linux') || p.includes('Build') || p.includes('wv') || p.length <= 2) continue
        if (p.length < 30) {
          device = p
          break
        }
      }
    }
    if (device === 'Unknown') {
      device = ua.includes('Mobile') ? 'Android Mobile' : 'Android Tablet'
    }
  } else if (ua.includes('Windows')) {
    device = 'PC'
  } else if (ua.includes('Macintosh')) {
    device = 'Mac'
  } else if (ua.includes('Linux')) {
    device = 'Linux PC'
  } else if (uaLower.includes('curl') || uaLower.includes('python') || uaLower.includes('axios') || uaLower.includes('postman') || uaLower.includes('bot') || ua.includes('Go-http-client')) {
    device = 'Server / Bot'
  }

  return { browser, os, cpu, device }
}

function parseCLFTimestamp(value: string): string | null {
  const months: Record<string, number> = {
    Jan: 1,
    Feb: 2,
    Mar: 3,
    Apr: 4,
    May: 5,
    Jun: 6,
    Jul: 7,
    Aug: 8,
    Sep: 9,
    Oct: 10,
    Nov: 11,
    Dec: 12,
  }
  const match = value.match(/^(\d{2})\/([A-Za-z]{3})\/(\d{4}):(\d{2}):(\d{2}):(\d{2}) ([+-]\d{4})$/)
  if (!match) {
    const parsed = Date.parse(value)
    return Number.isNaN(parsed) ? null : new Date(parsed).toISOString()
  }
  const [, day, monthName, year, hour, minute, second, offset] = match
  const month = months[monthName]
  if (!month) return null
  const isoLike = `${year}-${String(month).padStart(2, '0')}-${day}T${hour}:${minute}:${second}${offset.slice(0, 3)}:${offset.slice(3)}`
  const parsed = Date.parse(isoLike)
  return Number.isNaN(parsed) ? null : new Date(parsed).toISOString()
}

export function getMethodStyles(method: string) {
  switch (method?.toUpperCase()) {
    case 'GET': return 'bg-blue-500/15 text-blue-700 dark:text-blue-400 border-blue-500/20'
    case 'POST': return 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 border-emerald-500/20'
    case 'PUT': return 'bg-amber-500/15 text-amber-700 dark:text-amber-400 border-amber-500/20'
    case 'DELETE': return 'bg-rose-500/15 text-rose-700 dark:text-rose-400 border-rose-500/20'
    case 'PATCH': return 'bg-purple-500/15 text-purple-700 dark:text-purple-400 border-purple-500/20'
    default: return 'bg-muted text-muted-foreground border-transparent'
  }
}

export function getStatusVariant(status: number): 'success' | 'info' | 'warning' | 'destructive' | 'secondary' {
  if (status >= 200 && status < 300) return 'success'
  if (status >= 300 && status < 400) return 'info'
  if (status >= 400 && status < 500) return 'warning'
  if (status >= 500) return 'destructive'
  return 'secondary'
}

export function parseTraefikLog(line: string) {
  try {
    let cleanLine = line.trim()
    let dockerTime: string | null = null

    // Check for Docker timestamp prefix (e.g. "2026-10-01T08:42:00.123456789Z ")
    const dockerTsMatch = cleanLine.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2}))\s+(.*)$/)
    if (dockerTsMatch) {
      dockerTime = dockerTsMatch[1]
      cleanLine = dockerTsMatch[2].trim()
    }

    if (cleanLine.startsWith('{')) {
      const d = JSON.parse(cleanLine)
      const rawDuration = d.OriginDuration ?? d.origin_duration ?? d.Duration ?? d.duration
      const durationNumber = rawDuration == null || rawDuration === '' ? NaN : Number(rawDuration)
      const durationMs = Number.isFinite(durationNumber) ? durationNumber / 1_000_000 : undefined
      const statusRaw = d.DownstreamStatus ?? d.status ?? d.downstream_Status ?? d.OriginStatus
      const statusNum = statusRaw != null && statusRaw !== '' ? Number(statusRaw) : null
      return {
        ...d,
        Duration: durationMs,
        t: d.time || d.Time || d.StartLocal || d.StartUTC || d.t || dockerTime,
        ip: d.ClientHost || d.ClientAddr || d.client_ip || d.ip,
        method: d.RequestMethod || d.method || d.request_Method,
        path: d.RequestPath || d.path || d.request_Path,
        host: d.RequestHost || d.request_Host || d.host,
        ua: d.UserAgent || d["request_User-Agent"] || d.user_agent || d.ua,
        status: Number.isFinite(statusNum) ? statusNum : null,
        duration: durationMs,
        service: d.ServiceName || d.service,
        msg: d.msg || d.message || cleanLine
      }
    }
    const clf = cleanLine.match(/^(\S+) \S+ \S+ \[(.*?)\] "(\S+) (\S+) \S+" (\d+) (\d+)/)
    if (clf) {
      return {
        ip: clf[1],
        t: parseCLFTimestamp(clf[2]) || dockerTime,
        method: clf[3],
        path: clf[4],
        status: parseInt(clf[5], 10),
        msg: cleanLine
      }
    }
  } catch (e) {}
  return { msg: line, t: null, ip: null, method: null, path: null, status: null }
}

export function groupStatusCodes(statusCodes: { name: string; value: number }[]) {
  if (!statusCodes) return []
  const groups: Record<string, number> = { '2xx': 0, '3xx': 0, '4xx': 0, '5xx': 0 }
  statusCodes.forEach((c) => {
    const firstDigit = c.name.charAt(0)
    const groupName = `${firstDigit}xx`
    if (groups[groupName] !== undefined) {
      groups[groupName] += c.value
    }
  })
  return Object.entries(groups)
    .filter(([_, value]) => value > 0)
    .map(([name, value]) => {
      let fill = 'hsl(var(--muted))'
      if (name.startsWith('2')) fill = 'hsl(var(--success))'
      else if (name.startsWith('3')) fill = 'hsl(var(--info))'
      else if (name.startsWith('4')) fill = 'hsl(var(--warning))'
      else if (name.startsWith('5')) fill = 'hsl(var(--destructive))'
      return { name, value, fill }
    })
}
