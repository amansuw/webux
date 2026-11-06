package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/brendan4linux/webux/internal/config"
)

// CloudflareHandler proxies Cloudflare GraphQL analytics for the dashboard.
type CloudflareHandler struct {
	cfg *config.CloudflareConfig

	mu        sync.Mutex
	cached    map[string]any
	cachedAt  time.Time
	cacheTTL  time.Duration
}

func NewCloudflareHandler(cfg *config.CloudflareConfig) *CloudflareHandler {
	return &CloudflareHandler{
		cfg:      cfg,
		cacheTTL: 2 * time.Minute,
	}
}

func (h *CloudflareHandler) configured() bool {
	return h.cfg != nil && h.cfg.APIToken != "" && h.cfg.ZoneID != ""
}

// Overview handles GET /api/cloudflare/overview — last-24h zone analytics.
func (h *CloudflareHandler) Overview(w http.ResponseWriter, r *http.Request) {
	if !h.configured() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{
			"configured": false,
			"error":      "Cloudflare analytics not configured (set cloudflare.api_token + zone_id)",
		})
		return
	}

	h.mu.Lock()
	if h.cached != nil && time.Since(h.cachedAt) < h.cacheTTL {
		out := h.cached
		h.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
		return
	}
	h.mu.Unlock()

	data, err := h.fetchOverview()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	data["configured"] = true
	data["zone_name"] = h.cfg.ZoneName
	data["zone_id"] = h.cfg.ZoneID
	data["fetched_at"] = time.Now().UTC().Format(time.RFC3339)

	h.mu.Lock()
	h.cached = data
	h.cachedAt = time.Now()
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *CloudflareHandler) fetchOverview() (map[string]any, error) {
	end := time.Now().UTC().Truncate(time.Hour)
	start := end.Add(-24 * time.Hour)
	dayStart := end.Add(-48 * time.Hour).Format("2006-01-02")

	query := fmt.Sprintf(`
query {
  viewer {
    zones(filter: {zoneTag: "%s"}) {
      series: httpRequests1hGroups(
        limit: 24
        orderBy: [datetime_ASC]
        filter: {datetime_geq: "%s", datetime_lt: "%s"}
      ) {
        dimensions { datetime }
        sum {
          requests
          bytes
          cachedBytes
          cachedRequests
          pageViews
          threats
          responseStatusMap { edgeResponseStatus requests }
        }
        uniq { uniques }
      }
      days: httpRequests1dGroups(
        limit: 2
        orderBy: [date_DESC]
        filter: {date_geq: "%s"}
      ) {
        dimensions { date }
        sum {
          countryMap { clientCountryName requests bytes threats }
        }
      }
    }
  }
}`, h.cfg.ZoneID,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		dayStart,
	)

	body, _ := json.Marshal(map[string]string{"query": query})
	req, err := http.NewRequest(http.MethodPost, "https://api.cloudflare.com/client/v4/graphql", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.APIToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cloudflare graphql HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	var parsed struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Data struct {
			Viewer struct {
				Zones []struct {
					Series []struct {
						Dimensions struct {
							Datetime string `json:"datetime"`
						} `json:"dimensions"`
						Sum struct {
							Requests        int64 `json:"requests"`
							Bytes           int64 `json:"bytes"`
							CachedBytes     int64 `json:"cachedBytes"`
							CachedRequests  int64 `json:"cachedRequests"`
							PageViews       int64 `json:"pageViews"`
							Threats         int64 `json:"threats"`
							ResponseStatusMap []struct {
								EdgeResponseStatus int   `json:"edgeResponseStatus"`
								Requests           int64 `json:"requests"`
							} `json:"responseStatusMap"`
						} `json:"sum"`
						Uniq struct {
							Uniques int64 `json:"uniques"`
						} `json:"uniq"`
					} `json:"series"`
					Days []struct {
						Dimensions struct {
							Date string `json:"date"`
						} `json:"dimensions"`
						Sum struct {
							CountryMap []struct {
								ClientCountryName string `json:"clientCountryName"`
								Requests          int64  `json:"requests"`
								Bytes             int64  `json:"bytes"`
								Threats           int64  `json:"threats"`
							} `json:"countryMap"`
						} `json:"sum"`
					} `json:"days"`
				} `json:"zones"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("cloudflare graphql: %s", parsed.Errors[0].Message)
	}
	if len(parsed.Data.Viewer.Zones) == 0 {
		return nil, fmt.Errorf("no zone data returned")
	}
	z := parsed.Data.Viewer.Zones[0]

	var (
		totalReq, totalBytes, cachedReq, cachedBytes, pageViews, threats, uniques int64
		series []map[string]any
		status = map[string]int64{}
	)
	for _, g := range z.Series {
		s := g.Sum
		totalReq += s.Requests
		totalBytes += s.Bytes
		cachedReq += s.CachedRequests
		cachedBytes += s.CachedBytes
		pageViews += s.PageViews
		threats += s.Threats
		uniques += g.Uniq.Uniques
		stHour := map[string]int64{"2xx": 0, "3xx": 0, "4xx": 0, "5xx": 0, "other": 0}
		for _, m := range s.ResponseStatusMap {
			bucket := statusBucket(m.EdgeResponseStatus)
			status[bucket] += m.Requests
			stHour[bucket] += m.Requests
		}
		uncached := s.Requests - s.CachedRequests
		if uncached < 0 {
			uncached = 0
		}
		series = append(series, map[string]any{
			"t":               g.Dimensions.Datetime,
			"requests":        s.Requests,
			"bytes":           s.Bytes,
			"page_views":      s.PageViews,
			"cached_requests": s.CachedRequests,
			"uncached":        uncached,
			"s2xx":            stHour["2xx"],
			"s3xx":            stHour["3xx"],
			"s4xx":            stHour["4xx"],
			"s5xx":            stHour["5xx"],
		})
	}

	hitRate := 0.0
	if totalReq > 0 {
		hitRate = float64(cachedReq) / float64(totalReq) * 100
	}
	err5xx := 0.0
	if totalReq > 0 {
		err5xx = float64(status["5xx"]) / float64(totalReq) * 100
	}

	// countries from most recent day group
	countries := []map[string]any{}
	if len(z.Days) > 0 {
		cmap := append([]struct {
			ClientCountryName string `json:"clientCountryName"`
			Requests          int64  `json:"requests"`
			Bytes             int64  `json:"bytes"`
			Threats           int64  `json:"threats"`
		}{}, z.Days[0].Sum.CountryMap...)
		sort.Slice(cmap, func(i, j int) bool { return cmap[i].Requests > cmap[j].Requests })
		limit := 8
		if len(cmap) < limit {
			limit = len(cmap)
		}
		for _, c := range cmap[:limit] {
			countries = append(countries, map[string]any{
				"country":  c.ClientCountryName,
				"requests": c.Requests,
				"bytes":    c.Bytes,
				"threats":  c.Threats,
			})
		}
	}

	statusList := []map[string]any{}
	for _, k := range []string{"2xx", "3xx", "4xx", "5xx"} {
		statusList = append(statusList, map[string]any{"code": k, "requests": status[k]})
	}

	return map[string]any{
		"window": "24h",
		"totals": map[string]any{
			"requests":         totalReq,
			"bytes":            totalBytes,
			"cached_requests":  cachedReq,
			"cached_bytes":     cachedBytes,
			"page_views":       pageViews,
			"threats":          threats,
			"uniques_sum":      uniques,
			"cache_hit_rate":   hitRate,
			"error_5xx_rate":   err5xx,
		},
		"series":    series,
		"status":    statusList,
		"countries": countries,
	}, nil
}

func statusBucket(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "other"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
