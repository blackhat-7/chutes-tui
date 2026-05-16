package model

import (
	"math"
	"strings"
	"sync"
)

const DefaultQuality = 50

var (
	qualityCache   = make(map[string]int)
	qualityCacheMu sync.RWMutex
)

func SetQualityCache(cache map[string]int) {
	qualityCacheMu.Lock()
	defer qualityCacheMu.Unlock()
	qualityCache = cache
}

func HasQualityCache() bool {
	qualityCacheMu.RLock()
	defer qualityCacheMu.RUnlock()
	return len(qualityCache) > 0
}

func GetQualityCache() map[string]int {
	qualityCacheMu.RLock()
	defer qualityCacheMu.RUnlock()
	out := make(map[string]int, len(qualityCache))
	for k, v := range qualityCache {
		out[k] = v
	}
	return out
}

type Chute struct {
	ChuteID                  string
	Name                     string
	UtilizationCurrent       float64
	Utilization5m            float64
	Utilization1h            float64
	RateLimitRatio5m         float64
	RateLimitRatio1h         float64
	TotalRequests1h          float64
	CompletedRequests1h      float64
	RateLimitedRequests1h    float64
	InstanceCount            int
	ActiveInstanceCount      int
	Scalable                 bool
	ActionTaken              string
	EffectiveMultiplier      float64
	TEE                      bool
}

func (c Chute) DisplayName() string {
	parts := strings.SplitN(c.Name, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return c.Name
}

func (c Chute) Quality() int {
	normed := Norm(c.Name)
	best, bestLen := DefaultQuality, 0

	qualityCacheMu.RLock()
	defer qualityCacheMu.RUnlock()

	for key, score := range qualityCache {
		if strings.Contains(normed, key) && len(key) > bestLen {
			best, bestLen = score, len(key)
		}
	}
	return best
}

func (c Chute) Availability() float64 {
	base := math.Max(0.0, 1.0-c.RateLimitRatio1h*0.65-c.Utilization1h*0.35)
	if c.TotalRequests1h == 0 {
		return math.Min(base, 0.8)
	}
	return base
}

func (c Chute) Score() float64 {
	return float64(c.Quality())/100.0*0.55 + c.Availability()*0.45
}

func (c Chute) IsUsable() bool {
	return c.ActiveInstanceCount > 0 && c.RateLimitRatio5m < 0.98
}

func Norm(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSep := false
	for _, ch := range strings.ToLower(s) {
		switch ch {
		case '-', '.', '_', '/', ' ', '\t', '\n':
			if !inSep {
				b.WriteByte('-')
				inSep = true
			}
		default:
			b.WriteRune(ch)
			inSep = false
		}
	}
	return strings.Trim(b.String(), "-")
}
