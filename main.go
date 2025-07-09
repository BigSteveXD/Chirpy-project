package main

import (
	"log"
	"net/http"
	"sync/atomic"
	"fmt"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) countHits(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hits: %d", hits)
}

func (cfg *apiConfig) resetHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}

func main() {
	//new http.ServeMux
	myServeMux := http.NewServeMux()

	apiCfg := &apiConfig{}

	myServeMux.Handle("/app/", apiCfg.middlewareMetricsInc( http.StripPrefix("/app", http.FileServer(http.Dir("."))) ))//fileserver is a handler

	myServeMux.Handle("/metrics", http.HandlerFunc(apiCfg.countHits))//handle the wrap(handler)
	myServeMux.Handle("/reset", http.HandlerFunc(apiCfg.resetHits))

	//readiness endpoint
	myServeMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	//custom server
	myServer := &http.Server{
		Addr: ":8080",
		Handler: myServeMux,
		//ReadTimeout: 10 * time.Second,
		//WriteTimeout: 10 * time.Second,
		//MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(myServer.ListenAndServe())
}
