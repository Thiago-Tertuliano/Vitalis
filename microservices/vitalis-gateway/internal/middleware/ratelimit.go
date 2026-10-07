package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/httpx"
	"golang.org/x/time/rate"
)

// RateLimiter: token bucket por IP, em memória (Redis distribuído fica para depois).
type RateLimiter struct {
	rps   rate.Limit
	burst int

	mu      sync.Mutex
	clients map[string]*visitor
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{rps: rate.Limit(rps), burst: burst, clients: map[string]*visitor{}}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.limiterFor(clientIP(r)).Allow() {
			w.Header().Set("Retry-After", "1")
			httpx.WriteError(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "muitas requisições, tente novamente")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, ok := rl.clients[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.clients[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// cleanup remove IPs inativos para o mapa não crescer para sempre.
func (rl *RateLimiter) cleanup() {
	for range time.Tick(time.Minute) {
		rl.mu.Lock()
		for ip, v := range rl.clients {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
