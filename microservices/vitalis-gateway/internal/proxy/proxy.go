package proxy

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/httpx"
)

// Registry guarda um reverse proxy por serviço interno.
type Registry struct {
	proxies map[string]*httputil.ReverseProxy
}

func NewRegistry(upstreams map[string]string, timeout time.Duration, log *slog.Logger) (*Registry, error) {
	// Um Transport compartilhado reaproveita conexões keep-alive com os serviços.
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: timeout,
	}

	reg := &Registry{proxies: make(map[string]*httputil.ReverseProxy, len(upstreams))}
	for name, raw := range upstreams {
		target, err := url.Parse(raw)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return nil, fmt.Errorf("upstream %s inválido: %q", name, raw)
		}
		reg.proxies[name] = &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(target)  // troca host/scheme mantendo path e query
				pr.SetXForwarded() // X-Forwarded-For/Host/Proto para o serviço saber a origem
			},
			Transport: transport,
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				log.Error("upstream indisponível", "service", name, "path", r.URL.Path, "err", err)
				httpx.WriteError(w, r, http.StatusBadGateway, "BAD_GATEWAY", "serviço "+name+" indisponível")
			},
		}
	}
	return reg, nil
}

func (r *Registry) Get(name string) (*httputil.ReverseProxy, bool) {
	p, ok := r.proxies[name]
	return p, ok
}
