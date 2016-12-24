package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	hport := flag.String("h", ":80", "hostport")
	flag.Parse()
	http.Handle("/", http.FileServer(http.Dir("./")))
	log.Fatal(http.ListenAndServe(*hport, nil))
}
