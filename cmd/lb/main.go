package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync/atomic"
)

type roundRobin struct {
	targets []*url.URL
	counter atomic.Uint64
}

func (rr *roundRobin) next() *url.URL {
	i := rr.counter.Add(1) % uint64(len(rr.targets))
	return rr.targets[i]
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	lbPort := getEnv("LB_PORT", "9999")
	api01 := getEnv("API01_URL", "http://api01:8080")
	api02 := getEnv("API02_URL", "http://api02:8081")

	url1, err := url.Parse(api01)
	if err != nil {
		log.Fatalf("invalid API01_URL: %v", err)
	}
	url2, err := url.Parse(api02)
	if err != nil {
		log.Fatalf("invalid API02_URL: %v", err)
	}

	rr := &roundRobin{targets: []*url.URL{url1, url2}}

	proxy := httputil.ReverseProxy{
		Director: func(r *http.Request) {
			target := rr.next()
			r.URL.Scheme = target.Scheme
			r.URL.Host = target.Host
			r.Header.Set("X-Real-IP", r.RemoteAddr)
			if r.Header.Get("X-Forwarded-For") == "" {
				r.Header.Set("X-Forwarded-For", r.RemoteAddr)
			}
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", proxy.ServeHTTP)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("LB listening on :%s", lbPort)
	log.Fatal(http.ListenAndServe(":"+lbPort, mux))
}
