package main

import (
	"flag"
	backend "staticbackend"
)

func main() {
	dbHost := flag.String("host", "localhost", "Hostname for mongodb")
	port := flag.String("port", "8099", "HTTP port to listen on")
	flag.Parse()

	backend.Start(*dbHost, *port)
}
