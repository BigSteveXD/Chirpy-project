package main

import (
	"log"
	"net/http"
)

func main() {
	//new http.ServeMux
	myServeMux := http.NewServeMux()

	myServeMux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))

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
