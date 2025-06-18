package main

import (
	"net/http"
)

func main() {
	//new http.ServeMux
	myServeMux := http.NewServeMux()

	myServeMux.Handle("/", http.FileServer(http.Dir(".")))

	//custom server
	myServer := &http.Server{
		Addr: ":8080",
		Handler: myServeMux,
		//ReadTimeout: 10 * time.Second,
		//WriteTimeout: 10 * time.Second,
		//MaxHeaderBytes: 1 << 20,
	}
	//log.Fatal(myServer.ListenAndServe())
	myServer.ListenAndServe()
}
