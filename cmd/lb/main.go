package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

type backend struct {
	url   *url.URL
	alive atomic.Bool
}

type roundRobin struct {
	backends []*backend
	counter  atomic.Uint64
}

func (rr *roundRobin) next() *backend {
	n := uint64(len(rr.backends))
	for i := uint64(0); i < n; i++ {
		idx := rr.counter.Add(1) % n
		be := rr.backends[idx]
		if be.alive.Load() {
			return be
		}
	}
	return rr.backends[0]
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func healthCheck(interval time.Duration, backends []*backend) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	client := &http.Client{Timeout: 2 * time.Second}
	for range ticker.C {
		for _, be := range backends {
			go func(be *backend) {
				u := *be.url
				u.Path = "/health"
				resp, err := client.Get(u.String())
				if err != nil || resp.StatusCode >= 500 {
					be.alive.Store(false)
				} else {
					be.alive.Store(true)
				}
				if resp != nil {
					resp.Body.Close()
				}
			}(be)
		}
	}
}

func main() {
	lbPort := getEnv("LB_PORT", "9999")
	api01 := getEnv("API01_URL", "http://api01:8080")
	api02 := getEnv("API02_URL", "http://api02:8081")

	parse := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		if err != nil {
			log.Fatalf("invalid URL %q: %v", raw, err)
		}
		return u
	}

	backends := []*backend{
		{url: parse(api01)},
		{url: parse(api02)},
	}
	for _, be := range backends {
		be.alive.Store(true)
	}

	rr := &roundRobin{backends: backends}

	transport := &http.Transport{
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   64,
		MaxConnsPerHost:       128,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 3 * time.Second,
		DisableCompression:    true,
	}

	proxy := &httputil.ReverseProxy{
		Transport: transport,
		Director: func(r *http.Request) {
			be := rr.next()
			r.URL.Scheme = be.url.Scheme
			r.URL.Host = be.url.Host
			r.Header.Set("X-Real-IP", r.RemoteAddr)
			if r.Header.Get("X-Forwarded-For") == "" {
				r.Header.Set("X-Forwarded-For", r.RemoteAddr)
			}
		},
	}

	go healthCheck(5*time.Second, backends)

	mux := http.NewServeMux()
	mux.HandleFunc("/", proxy.ServeHTTP)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:              ":" + lbPort,
		Handler:           mux,
		ReadTimeout:       3 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-done
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("LB listening on :%s", lbPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
