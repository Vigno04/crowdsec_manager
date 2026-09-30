package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"crowdsec-manager/internal/cache"
	"crowdsec-manager/internal/config"
	"crowdsec-manager/internal/database"
	"crowdsec-manager/internal/docker"
	"crowdsec-manager/internal/geoip"
	"crowdsec-manager/internal/logger"
	"crowdsec-manager/internal/logs/aggregate"
	"crowdsec-manager/internal/models"

	"github.com/gin-gonic/gin"
)

type dashboardLogReader interface {
	ExecCommand(containerName string, cmd []string) (string, error)
	GetContainerLogs(containerName string, tail string) (string, error)
}

type serviceDashboardHandlerInput struct {
	Reader   dashboardLogReader
	Database *database.Database
	Config   *config.Config
	Geo      *geoip.Resolver
	Cache    *cache.TTLCache
}

type traefikLogReadInput struct {
	Reader   dashboardLogReader
	Database *database.Database
	Config   *config.Config
	Tail     string
}

// rangeTailMap maps each preset to a max-tail size that bounds memory.
// Anything older than the timestamp cutoff is filtered downstream.
var rangeTailMap = map[models.DashboardRange]string{
	models.Range5m:  "2000",
	models.Range1h:  "10000",
	models.Range6h:  "30000",
	models.Range24h: "60000",
	models.Range7d:  "200000",
	models.RangeAll: "500000",
}

func rangeDuration(rng models.DashboardRange) time.Duration {
	switch rng {
	case models.Range5m:
		return 5 * time.Minute
	case models.Range1h:
		return time.Hour
	case models.Range6h:
		return 6 * time.Hour
	case models.Range24h:
		return 24 * time.Hour
	case models.Range7d:
		return 7 * 24 * time.Hour
	case models.RangeAll:
		// Use a 10-year cap as a practical "all time" ceiling to bound
		// storage, memory, and retention costs without pretending history is infinite.
		return 3650 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func parseDashboardRange(raw string) (models.DashboardRange, bool) {
	r := models.DashboardRange(raw)
	if _, ok := rangeTailMap[r]; ok {
		return r, true
	}
	return "", false
}

// geoAdapter exposes *geoip.Resolver as aggregate.GeoLookup without the
// aggregate package importing geoip (which would create a cycle in tests).
type geoAdapter struct{ r *geoip.Resolver }

func (a geoAdapter) Lookup(ip string) (aggregate.Location, bool) {
	if a.r == nil {
		return aggregate.Location{}, false
	}
	loc, ok := a.r.Lookup(ip)
	if !ok {
		return aggregate.Location{}, false
	}
	return aggregate.Location{Country: loc.Country, Lat: loc.Lat, Lng: loc.Lng}, true
}

// AnalyzeServiceDashboard returns a service-shaped dashboard payload.
// Supports :service in {traefik, crowdsec}.
func AnalyzeServiceDashboard(dockerClient *docker.Client, db *database.Database, cfg *config.Config, geo *geoip.Resolver, ttlCache ...*cache.TTLCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler := analyzeServiceDashboardWithReader(serviceDashboardHandlerInput{
			Reader:   resolveDockerClient(c, dockerClient),
			Database: db,
			Config:   cfg,
			Geo:      geo,
			Cache:    optionalCache(ttlCache),
		})
		handler(c)
	}
}

func analyzeServiceDashboardWithReader(input serviceDashboardHandlerInput) gin.HandlerFunc {
	adapter := geoAdapter{r: input.Geo}
	return func(c *gin.Context) {
		service := c.Param("service")

		rngRaw := c.DefaultQuery("range", string(models.Range1h))
		rng, ok := parseDashboardRange(rngRaw)
		if !ok {
			c.JSON(http.StatusBadRequest, models.Response{
				Success: false,
				Error:   fmt.Sprintf("invalid range %q (allowed: 5m,1h,6h,24h,7d,all)", rngRaw),
			})
			return
		}

		if service != "traefik" && service != "crowdsec" {
			c.JSON(http.StatusBadRequest, models.Response{
				Success: false,
				Error:   fmt.Sprintf("unsupported service %q (expected traefik or crowdsec)", service),
			})
			return
		}

		if !requireLogProcessingEnabled(c, input.Database) {
			return
		}

		tail := rangeTailMap[rng]
		cacheKey := serviceDashboardCacheKey(c, service)
		if input.Cache != nil {
			if cached, ok := input.Cache.Get(cacheKey); ok {
				c.JSON(http.StatusOK, models.Response{Success: true, Data: cached})
				return
			}
		}

		now := time.Now().UTC()
		since := now.Add(-rangeDuration(rng))

		switch service {
		case "traefik":
			systemStats := aggregate.GetSystemStats()
			rawLogs, warning, err := readTraefikLogs(traefikLogReadInput{
				Reader:   input.Reader,
				Database: input.Database,
				Config:   input.Config,
				Tail:     tail,
			})
			if err != nil {
				logger.Warn("failed to read traefik logs for dashboard", "error", err)
				c.JSON(http.StatusInternalServerError, models.Response{
					Success: false,
					Error:   fmt.Sprintf("failed to read traefik logs: %v", err),
				})
				return
			}
			data := aggregate.BucketTraefikRaw(rawLogs, since, now, rng, adapter, systemStats)
			data.Warning = warning
			if input.Cache != nil {
				input.Cache.Set(cacheKey, data, serviceDashboardCacheTTL)
			}
			c.JSON(http.StatusOK, models.Response{Success: true, Data: data})

		case "crowdsec":
			parser := docker.NewLogParser()
			rawLogs, err := input.Reader.GetContainerLogs(input.Config.CrowdsecContainerName, tail)
			if err != nil {
				logger.Warn("failed to read crowdsec logs for dashboard", "error", err)
				c.JSON(http.StatusInternalServerError, models.Response{
					Success: false,
					Error:   fmt.Sprintf("failed to read crowdsec logs: %v", err),
				})
				return
			}
			entries := parser.Parse(rawLogs, "crowdsec")
			data := aggregate.BucketCrowdSec(entries, since, now, rng, adapter)
			if input.Cache != nil {
				input.Cache.Set(cacheKey, data, serviceDashboardCacheTTL)
			}
			c.JSON(http.StatusOK, models.Response{Success: true, Data: data})

		}
	}
}

// readTraefikLogs prefers reading directly from the mounted access log file on the local filesystem
// (instant, zero Docker IPC overhead); falls back to container exec tail, and finally container logs.
func readTraefikLogs(input traefikLogReadInput) (string, string, error) {
	logPath := ""
	if input.Database != nil {
		settings, _ := input.Database.GetSettings()
		logPath = settings.TraefikAccessLog
	}
	if logPath == "" {
		logPath = input.Config.TraefikAccessLog
	}

	fallbackWarning := "traefik log read failed, using slower docker logs, check mounts"

	if logPath != "" {
		// Fast-path: if the access log is directly accessible on the local filesystem (e.g. mounted volume),
		// read it directly via reverse seeking to avoid Docker exec latency and IPC overhead.
		if fi, err := os.Stat(logPath); err == nil && !fi.IsDir() {
			if logs, err := readLocalLogTail(logPath, input.Tail); err == nil {
				return logs, "", nil
			} else {
				logger.Debug("failed to read local traefik access log; falling back to container exec", "error", err)
			}
		}

		tailCount, _ := strconv.Atoi(input.Tail)
		if tailCount <= 0 {
			tailCount = 10000
		}
		// Since Traefik access log entries average ~2.5 KB, BusyBox tail -n on large files
		// scans sequentially and can lock up CPU for 30s. Using tail -c utilizes lseek (<1ms).
		byteCount := tailCount * 2500
		fastCmd := []string{
			"sh", "-c",
			fmt.Sprintf("test -f %s && tail -c %d %s 2>/dev/null | tail -n %d", strconv.Quote(logPath), byteCount, strconv.Quote(logPath), tailCount),
		}

		if logs, err := input.Reader.ExecCommand(input.Config.TraefikContainerName, fastCmd); err == nil {
			return logs, fallbackWarning, nil
		}

		// Fallback to standard tail -n if sh/pipe fails
		if logs, err := input.Reader.ExecCommand(input.Config.TraefikContainerName, []string{"tail", "-n", input.Tail, logPath}); err == nil {
			return logs, fallbackWarning, nil
		} else {
			logger.Debug("traefik access log file unreadable; falling back to container logs", "error", err)
		}
	}
	logs, err := input.Reader.GetContainerLogs(input.Config.TraefikContainerName, input.Tail)
	return logs, fallbackWarning, err
}

// readLocalLogTail reads the last tail lines from a local file efficiently without loading
// the entire file into memory by seeking backwards from the end of the file in chunks.
func readLocalLogTail(path string, tailStr string) (string, error) {
	tailCount, err := strconv.Atoi(tailStr)
	if err != nil || tailCount <= 0 {
		tailCount = 10000
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return "", nil
	}

	const chunkSize = 64 * 1024
	buf := make([]byte, chunkSize)
	var lineCount int
	offset := fileSize

	// Scan backwards from EOF to find the byte offset corresponding to the requested tail line count
	for offset > 0 && lineCount < tailCount {
		readSize := int64(chunkSize)
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize

		_, err := file.Seek(offset, io.SeekStart)
		if err != nil {
			break
		}

		n, err := io.ReadFull(file, buf[:readSize])
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			break
		}

		chunk := buf[:n]
		for i := len(chunk) - 1; i >= 0; i-- {
			if chunk[i] == '\n' {
				lineCount++
				if lineCount > tailCount {
					offset += int64(i + 1)
					break
				}
			}
		}
	}

	if offset < 0 {
		offset = 0
	}

	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
