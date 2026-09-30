import { describe, expect, it } from 'vitest'
import { parseTraefikLog, parseUserAgent } from '@/lib/utils'

describe('parseTraefikLog', () => {
  it('normalizes Traefik JSON durations from nanoseconds to milliseconds', () => {
    const parsed = parseTraefikLog('{"Duration":12000000,"RequestMethod":"GET","RequestPath":"/health","DownstreamStatus":200}')

    expect(parsed.duration).toBe(12)
    expect(parsed.Duration).toBe(12)
  })

  it('normalizes CLF timestamps to ISO strings', () => {
    const parsed = parseTraefikLog('1.2.3.4 - - [07/May/2026:11:55:00 +0000] "GET /health HTTP/1.1" 200 100')

    expect(parsed.t).toBe('2026-05-07T11:55:00.000Z')
    expect(Number.isNaN(new Date(parsed.t as string).getTime())).toBe(false)
  })

  it('prioritizes time over StartUTC for long-lived connections', () => {
    const parsed = parseTraefikLog('{"StartUTC":"2026-09-19T17:20:32Z","time":"2026-09-29T11:13:48Z","Duration":841995601780913}')

    expect(parsed.t).toBe('2026-09-29T11:13:48Z')
  })
})

describe('parseUserAgent', () => {
  it('correctly parses full desktop Windows Chrome UA', () => {
    const info = parseUserAgent('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36')
    expect(info.browser).toBe('Chrome 120.0.0.0')
    expect(info.os).toBe('Windows 10/11')
    expect(info.cpu).toBe('x86 64-bit')
    expect(info.device).toBe('PC')
  })

  it('correctly parses simplified browser names from backend', () => {
    expect(parseUserAgent('Chrome').browser).toBe('Chrome')
    expect(parseUserAgent('Firefox').browser).toBe('Firefox')
    expect(parseUserAgent('Edge').browser).toBe('Edge')
    expect(parseUserAgent('Safari').browser).toBe('Safari')
  })

  it('correctly parses mobile iPhone UA', () => {
    const info = parseUserAgent('Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Mobile/15E148 Safari/604.1')
    expect(info.browser).toBe('Safari 604.1')
    expect(info.os).toBe('iOS 17.4.1')
    expect(info.device).toBe('iPhone')
  })

  it('correctly parses curl and command line tools', () => {
    const info = parseUserAgent('curl/8.4.0')
    expect(info.browser).toBe('Curl')
    expect(info.device).toBe('Server / Bot')
  })
})

