package main

import (
	//"fmt"
	"net/http"
)

func main() {
	//fmt.Println("hello")
	
	//new http.ServeMux
	//myServeMux = http.ServeMux("localhost:8080")
	myServeMux := http.NewServeMux()

	//create http.Server
	//myServer = new http.Server()

	//custom server
	//
	myServer := &http.Server{
		Addr: ":8080",
		Handler: myServeMux,
		//ReadTimeout: 10 * time.Second,
		//WriteTimeout: 10 * time.Second,
		//MaxHeaderBytes: 1 << 20,
	}
	//log.Fatal(myServer.ListenAndServe())
	myServer.ListenAndServe()
	//
}
