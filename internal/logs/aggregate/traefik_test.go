package aggregate

import (
	"strings"
	"testing"
	"time"

	"crowdsec-manager/internal/docker"
	"crowdsec-manager/internal/models"
)

type fakeGeo struct {
	hits map[string]Location
}

func (f fakeGeo) Lookup(ip string) (Location, bool) {
	loc, ok := f.hits[ip]
	return loc, ok
}

func testSystemStats() *models.SystemStats {
	return &models.SystemStats{}
}

func parseEntries(t *testing.T, raw string) []docker.StructuredLogEntry {
	t.Helper()
	parser := docker.NewLogParser()
	return parser.Parse(raw, "traefik")
}

func TestBucketTraefik_EmptyInput(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	d := BucketTraefik(nil, now.Add(-time.Hour), now, models.Range1h, fakeGeo{}, testSystemStats())
	if d.Format != "clf" {
		t.Fatalf("default format should be clf when nothing parses as JSON; got %q", d.Format)
	}
	if d.TotalRequests != 0 || d.UniqueIPs != 0 || d.ErrorRate != 0 {
		t.Fatalf("expected zero counters; got %+v", d)
	}
	if d.Series == nil || d.StatusCodes == nil || d.Methods == nil ||
		d.TopIPs == nil || d.TopHosts == nil || d.TopRouters == nil ||
		d.SlowestEndpoints == nil || d.TLSVersions == nil || d.RecentErrors == nil ||
		d.Browsers == nil || d.OperatingSystems == nil || d.Processors == nil || d.Devices == nil {
		t.Fatalf("nil slices must be replaced with empty slices for JSON serialisation: %+v", d)
	}
}

func TestBucketTraefik_CLFAggregates(t *testing.T) {
	logs := strings.Join([]string{
		// CLF: ip - - [time] "METHOD path HTTP" status size
		`1.2.3.4 - - [07/May/2026:11:55:00 +0000] "GET /a HTTP/1.1" 200 100`,
		`1.2.3.4 - - [07/May/2026:11:56:00 +0000] "GET /a HTTP/1.1" 200 100`,
		`5.6.7.8 - - [07/May/2026:11:57:00 +0000] "POST /b HTTP/1.1" 500 200`,
		`9.9.9.9 - - [07/May/2026:11:58:00 +0000] "GET /c HTTP/1.1" 404 50`,
	}, "\n")
	entries := parseEntries(t, logs)
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	geo := fakeGeo{hits: map[string]Location{
		"5.6.7.8": {Country: "DE", Lat: 51, Lng: 9},
	}}
	d := BucketTraefik(entries, now.Add(-time.Hour), now, models.Range1h, geo, testSystemStats())

	if d.Format != "clf" {
		t.Fatalf("expected CLF format, got %q", d.Format)
	}
	if d.TotalRequests != 4 {
		t.Fatalf("expected 4 requests, got %d", d.TotalRequests)
	}
	if d.UniqueIPs != 3 {
		t.Fatalf("expected 3 unique IPs, got %d", d.UniqueIPs)
	}
	// 1 5xx + 1 4xx out of 4 = 0.5
	if d.ErrorRate != 0.5 {
		t.Fatalf("expected error rate 0.5, got %v", d.ErrorRate)
	}
	if d.AvgDurationMs != nil {
		t.Fatalf("CLF mode must not produce avg duration; got %v", *d.AvgDurationMs)
	}

	// Status codes summed
	codeCounts := map[string]int{}
	for _, kv := range d.StatusCodes {
		codeCounts[kv.Name] = kv.Value
	}
	if codeCounts["200"] != 2 || codeCounts["404"] != 1 || codeCounts["500"] != 1 {
		t.Fatalf("unexpected status counts: %+v", codeCounts)
	}

	// Methods
	methodCounts := map[string]int{}
	for _, kv := range d.Methods {
		methodCounts[kv.Name] = kv.Value
	}
	if methodCounts["GET"] != 3 || methodCounts["POST"] != 1 {
		t.Fatalf("unexpected method counts: %+v", methodCounts)
	}

	// Top IPs sorted by count desc; 1.2.3.4 first
	if len(d.TopIPs) == 0 || d.TopIPs[0].IP != "1.2.3.4" || d.TopIPs[0].Count != 2 {
		t.Fatalf("top IP wrong: %+v", d.TopIPs)
	}
	// GeoIP populated for 5.6.7.8
	for _, ip := range d.TopIPs {
		if ip.IP == "5.6.7.8" && ip.Country != "DE" {
			t.Fatalf("expected DE for 5.6.7.8, got %+v", ip)
		}
	}

	// Recent errors should include 4xx and 5xx, sorted recent-first
	if len(d.RecentErrors) != 2 {
		t.Fatalf("expected 2 recent errors, got %d", len(d.RecentErrors))
	}
	if d.RecentErrors[0].Status < 400 {
		t.Fatalf("recent errors must only contain 4xx/5xx; got %+v", d.RecentErrors)
	}
}

func TestBucketTraefik_FiltersByCutoff(t *testing.T) {
	logs := strings.Join([]string{
		`1.2.3.4 - - [07/May/2026:09:00:00 +0000] "GET /old HTTP/1.1" 200 100`, // before cutoff
		`1.2.3.4 - - [07/May/2026:11:55:00 +0000] "GET /new HTTP/1.1" 200 100`, // inside
	}, "\n")
	entries := parseEntries(t, logs)
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	d := BucketTraefik(entries, now.Add(-time.Hour), now, models.Range1h, fakeGeo{}, testSystemStats())
	if d.TotalRequests != 1 {
		t.Fatalf("expected 1 request after cutoff filtering, got %d", d.TotalRequests)
	}
}

func TestBucketTraefik_JSONFormatPopulatesExtras(t *testing.T) {
	// Traefik JSON access log lines (one JSON object per line).
	logs := strings.Join([]string{
		`{"ClientHost":"1.2.3.4","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","RouterName":"router-a@docker","RequestPath":"/x","Duration":12000000,"StartUTC":"2026-05-07T11:50:00Z","TLSVersion":"1.3"}`,
		`{"ClientHost":"5.6.7.8","DownstreamStatus":500,"RequestMethod":"POST","RequestHost":"api.example.com","RouterName":"router-b@docker","RequestPath":"/y","Duration":40000000,"StartUTC":"2026-05-07T11:55:00Z","TLSVersion":"1.2"}`,
	}, "\n")
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	d := BucketTraefikRaw(logs, now.Add(-time.Hour), now, models.Range1h, fakeGeo{}, testSystemStats())

	if d.Format != "json" {
		t.Fatalf("expected json format when JSON lines present, got %q", d.Format)
	}
	if d.AvgDurationMs == nil {
		t.Fatalf("expected non-nil avg duration in JSON mode")
	}
	// Duration in nanoseconds: 12ms and 40ms -> avg 26ms
	if got := *d.AvgDurationMs; got < 25 || got > 27 {
		t.Fatalf("expected avg duration ~26ms, got %v", got)
	}
	if d.P95ResponseTimeMs == nil || *d.P95ResponseTimeMs < 38 || *d.P95ResponseTimeMs > 39 {
		t.Fatalf("expected interpolated p95 around 38.6ms, got %v", d.P95ResponseTimeMs)
	}
	if d.P99ResponseTimeMs == nil || *d.P99ResponseTimeMs < 39 || *d.P99ResponseTimeMs > 40 {
		t.Fatalf("expected interpolated p99 around 39.7ms, got %v", d.P99ResponseTimeMs)
	}
	if len(d.TopHosts) == 0 || len(d.TopRouters) == 0 {
		t.Fatalf("JSON mode should populate hosts/routers; got %+v / %+v", d.TopHosts, d.TopRouters)
	}
	if len(d.TLSVersions) == 0 {
		t.Fatalf("JSON mode should populate TLS versions")
	}
	if len(d.SlowestEndpoints) == 0 {
		t.Fatalf("JSON mode should populate slowest endpoints")
	}
}

func TestBucketTraefik_SlowestEndpointsExcludesStreaming(t *testing.T) {
	// A long-lived WebSocket stream (5 min) and a regular REST call (800 ms).
	// The stream's duration is connection lifetime, not latency, and must not
	// appear in SlowestEndpoints.
	logs := strings.Join([]string{
		`{"ClientHost":"1.2.3.4","DownstreamStatus":101,"RequestMethod":"GET","RequestHost":"example.com","RequestPath":"/api/logs/stream/traefik","Duration":300000000000,"StartUTC":"2026-05-07T11:50:00Z"}`,
		`{"ClientHost":"5.6.7.8","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","RequestPath":"/api/health","Duration":800000000,"StartUTC":"2026-05-07T11:55:00Z"}`,
		`{"ClientHost":"9.9.9.9","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","RequestPath":"/api/events/sse","Duration":120000000000,"StartUTC":"2026-05-07T11:56:00Z"}`,
		`{"ClientHost":"9.9.9.9","DownstreamStatus":101,"RequestMethod":"GET","RequestHost":"example.com","RequestPath":"/api/terminal/abc","Duration":60000000000,"StartUTC":"2026-05-07T11:57:00Z"}`,
	}, "\n")
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	d := BucketTraefikRaw(logs, now.Add(-time.Hour), now, models.Range1h, fakeGeo{}, testSystemStats())

	if d.Format != "json" {
		t.Fatalf("expected json format, got %q", d.Format)
	}
	if d.TotalRequests != 4 {
		t.Fatalf("streaming requests must still count toward totals; got %d", d.TotalRequests)
	}
	if len(d.SlowestEndpoints) != 1 {
		t.Fatalf("expected exactly one non-streaming endpoint; got %+v", d.SlowestEndpoints)
	}
	if d.SlowestEndpoints[0].Name != "/api/health" {
		t.Fatalf("expected /api/health as the only slowest endpoint; got %+v", d.SlowestEndpoints)
	}
	for _, kv := range d.SlowestEndpoints {
		if isStreamingPath(kv.Name) {
			t.Fatalf("streaming path %q must not appear in SlowestEndpoints", kv.Name)
		}
	}
}

func TestBucketTraefik_GranularityChoice(t *testing.T) {
	// 24h range should bucket by hour, not minute.
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	d := BucketTraefik(nil, now.Add(-24*time.Hour), now, models.Range24h, fakeGeo{}, testSystemStats())
	if d.Range != models.Range24h {
		t.Fatalf("range echoed wrong: %s", d.Range)
	}
}

func TestTraefikJSON_StartTime_PrioritizesTimeOverStartUTC(t *testing.T) {
	row := traefikJSON{
		StartUTC: "2026-09-19T17:20:32Z",
		Time:     "2026-09-29T11:13:48Z",
	}
	got := row.startTime()
	want, _ := time.Parse(time.RFC3339, "2026-09-29T11:13:48Z")
	if !got.Equal(want) {
		t.Fatalf("startTime() = %v, want %v", got, want)
	}
}

func TestBucketTraefik_UserAgentParsingAndAggregation(t *testing.T) {
	logs := strings.Join([]string{
		`{"ClientHost":"1.2.3.4","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","StartUTC":"2026-05-07T11:50:00Z","request_User-Agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"}`,
		`{"ClientHost":"1.2.3.4","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","StartUTC":"2026-05-07T11:51:00Z","request_User-Agent":"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Mobile/15E148 Safari/604.1"}`,
		`{"ClientHost":"5.6.7.8","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","StartUTC":"2026-05-07T11:52:00Z","request_User-Agent":"Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"}`,
		`{"ClientHost":"9.9.9.9","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","StartUTC":"2026-05-07T11:53:00Z","request_User-Agent":"curl/7.88.1"}`,
	}, "\n")

	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	d := BucketTraefikRaw(logs, now.Add(-time.Hour), now, models.Range1h, fakeGeo{}, testSystemStats())

	if d.Format != "json" {
		t.Fatalf("expected json format, got %q", d.Format)
	}

	// Browsers
	browserMap := map[string]int{}
	for _, b := range d.Browsers {
		browserMap[b.Name] = b.Value
	}
	if browserMap["Chrome"] != 2 {
		t.Errorf("expected 2 Chrome requests, got %d", browserMap["Chrome"])
	}
	if browserMap["Safari"] != 1 {
		t.Errorf("expected 1 Safari request, got %d", browserMap["Safari"])
	}
	if browserMap["Curl"] != 1 {
		t.Errorf("expected 1 Curl request, got %d", browserMap["Curl"])
	}

	// Operating Systems
	osMap := map[string]int{}
	for _, o := range d.OperatingSystems {
		osMap[o.Name] = o.Value
	}
	if osMap["Windows 10/11"] != 1 {
		t.Errorf("expected 1 Windows 10/11 request, got %d", osMap["Windows 10/11"])
	}
	if !strings.HasPrefix(d.OperatingSystems[0].Name, "Windows") && osMap["iOS 17.4.1"] != 1 {
		t.Errorf("expected iOS in OS map, got %+v", osMap)
	}
	if osMap["Android 14"] != 1 {
		t.Errorf("expected 1 Android 14 request, got %d", osMap["Android 14"])
	}

	// Processors / CPU
	cpuMap := map[string]int{}
	for _, c := range d.Processors {
		cpuMap[c.Name] = c.Value
	}
	if cpuMap["x86 64-bit"] != 1 {
		t.Errorf("expected 1 x86 64-bit, got %d", cpuMap["x86 64-bit"])
	}

	// Devices
	devMap := map[string]int{}
	for _, dev := range d.Devices {
		devMap[dev.Name] = dev.Value
	}
	if devMap["PC"] != 1 {
		t.Errorf("expected 1 PC, got %d", devMap["PC"])
	}
	if devMap["iPhone"] != 1 {
		t.Errorf("expected 1 iPhone, got %d", devMap["iPhone"])
	}
	if devMap["Pixel 8 Pro"] != 1 {
		t.Errorf("expected 1 Pixel 8 Pro, got %d", devMap["Pixel 8 Pro"])
	}
	if devMap["Server / Bot"] != 1 {
		t.Errorf("expected 1 Server / Bot, got %d", devMap["Server / Bot"])
	}
}

func TestBucketTraefik_PrefersOriginDurationOverDuration(t *testing.T) {
	// A request with OriginDuration (backend processing time) of 20ms
	// and Duration (including slow client transfer) of 5,000ms.
	logs := `{"ClientHost":"1.2.3.4","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","RequestPath":"/large-image.jpg","OriginDuration":20000000,"Duration":5000000000,"StartUTC":"2026-05-07T11:55:00Z"}`
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	d := BucketTraefikRaw(logs, now.Add(-time.Hour), now, models.Range1h, fakeGeo{}, testSystemStats())

	if d.AvgDurationMs == nil {
		t.Fatalf("expected non-nil avg duration")
	}
	// Expected ~20ms, NOT 5000ms
	if got := *d.AvgDurationMs; got < 19 || got > 21 {
		t.Fatalf("expected avg duration ~20ms from OriginDuration, got %v", got)
	}
	if len(d.SlowestEndpoints) != 1 || d.SlowestEndpoints[0].Value != 20 {
		t.Fatalf("expected slowest endpoint duration 20ms, got %+v", d.SlowestEndpoints)
	}
}

func TestBucketTraefik_ExcludesWebSocketsAndSSEFromLatencyMetrics(t *testing.T) {
	// 1 WebSocket (30s, status 101, socket.io), 1 SSE stream (10 min, status 200), and 1 normal REST (30ms).
	logs := strings.Join([]string{
		`{"ClientHost":"1.2.3.4","DownstreamStatus":101,"RequestMethod":"GET","RequestHost":"example.com","RequestPath":"/api/socket.io/?EIO=4&transport=websocket","Duration":30000000000,"StartUTC":"2026-05-07T11:50:00Z"}`,
		`{"ClientHost":"2.3.4.5","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","RequestPath":"/api/events/sse","Duration":600000000000,"StartUTC":"2026-05-07T11:51:00Z"}`,
		`{"ClientHost":"3.4.5.6","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","RequestPath":"/api/v1/photos","Duration":30000000,"StartUTC":"2026-05-07T11:55:00Z"}`,
	}, "\n")
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	d := BucketTraefikRaw(logs, now.Add(-time.Hour), now, models.Range1h, fakeGeo{}, testSystemStats())

	if d.TotalRequests != 3 {
		t.Fatalf("expected 3 total requests, got %d", d.TotalRequests)
	}
	if d.AvgDurationMs == nil {
		t.Fatalf("expected non-nil avg duration")
	}
	// Average latency should only be for /api/v1/photos (~30ms), NOT (30s + 600s + 0.03s)/3 = 210,010ms!
	if got := *d.AvgDurationMs; got < 29 || got > 31 {
		t.Fatalf("expected avg duration ~30ms, got %v", got)
	}
	if d.P95ResponseTimeMs == nil || *d.P95ResponseTimeMs < 29 || *d.P95ResponseTimeMs > 31 {
		t.Fatalf("expected P95 around 30ms, got %v", d.P95ResponseTimeMs)
	}
	if d.P99ResponseTimeMs == nil || *d.P99ResponseTimeMs < 29 || *d.P99ResponseTimeMs > 31 {
		t.Fatalf("expected P99 around 30ms, got %v", d.P99ResponseTimeMs)
	}
	// Slowest endpoints must only contain /api/v1/photos
	if len(d.SlowestEndpoints) != 1 || d.SlowestEndpoints[0].Name != "/api/v1/photos" {
		t.Fatalf("expected only /api/v1/photos in slowest endpoints; got %+v", d.SlowestEndpoints)
	}
}

func TestBucketTraefik_Excludes4xxTarpitsFromLatencyMetrics(t *testing.T) {
	// 1 legitimate request (25ms, status 200) and 1 tarpitted hacker probe (3000ms, status 403).
	logs := strings.Join([]string{
		`{"ClientHost":"1.2.3.4","DownstreamStatus":200,"RequestMethod":"GET","RequestHost":"example.com","RequestPath":"/api/data","Duration":25000000,"StartUTC":"2026-05-07T11:55:00Z"}`,
		`{"ClientHost":"9.9.9.9","DownstreamStatus":403,"RequestMethod":"GET","RequestHost":"example.com","RequestPath":"/.env","Duration":3000000000,"StartUTC":"2026-05-07T11:56:00Z"}`,
	}, "\n")
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	d := BucketTraefikRaw(logs, now.Add(-time.Hour), now, models.Range1h, fakeGeo{}, testSystemStats())

	if d.TotalRequests != 2 {
		t.Fatalf("expected 2 total requests, got %d", d.TotalRequests)
	}
	if d.ErrorRate != 0.5 {
		t.Fatalf("expected 50%% error rate, got %v", d.ErrorRate)
	}
	if d.AvgDurationMs == nil {
		t.Fatalf("expected non-nil avg duration")
	}
	// Average latency should only measure the legitimate request (~25ms), NOT (25ms + 3000ms)/2 = 1512ms!
	if got := *d.AvgDurationMs; got < 24 || got > 26 {
		t.Fatalf("expected avg duration ~25ms excluding 403 tarpit, got %v", got)
	}
	// Recent errors should still record the 403 probe with its duration
	if len(d.RecentErrors) != 1 || d.RecentErrors[0].Path != "/.env" || d.RecentErrors[0].DurationMs != 3000 {
		t.Fatalf("expected recent errors to preserve the 403 probe; got %+v", d.RecentErrors)
	}
}

