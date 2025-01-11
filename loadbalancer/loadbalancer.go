package loadbalancer

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

// Struct LoadBalancer para gerenciamento de servidores e estado
type LoadBalancer struct {
	servidores []*url.URL
	atual      uint32
}

// NewLoadBalancer inicializa um LoadBalancer com os URLs de servidor fornecidos
func NewLoadBalancer(URLservidores []string) *LoadBalancer {
	var servidores []*url.URL
	for _, URLservidor := range URLservidores {
		URLparseada, err := url.Parse(URLservidor)
		if err != nil {
			log.Fatalf("URL do servidor inválida: %s", err)
		}
		servidores = append(servidores, URLparseada)
	}

	return &LoadBalancer{
		servidores: servidores,
		atual:      0,
	}
}

// GetNextServer recupera o próximo URL do servidor usando round-robin
func (lb *LoadBalancer) GetNextServer() *url.URL {
	index := atomic.AddUint32(&lb.atual, 1)
	return lb.servidores[index%uint32(len(lb.servidores))]
}

// ServeHTTP encaminha a solicitação para o servidor selecionado
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := lb.GetNextServer()
	proxy := httputil.NewSingleHostReverseProxy(target)

	r.Host = target.Host
	r.URL.Host = target.Host
	r.URL.Scheme = target.Scheme

	// Atende a solicitação usando o proxy
	proxy.ServeHTTP(w, r)
}
