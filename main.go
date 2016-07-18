package main

import (
	"flag"
	"log"
)

var (
	Port string
	Addr string
	Host string
)

func init() {
	flag.StringVar(&Port, "p", "5300", "Port to listen to")
	flag.StringVar(&Port, "port", "5300", "Port to listen to")

	flag.StringVar(&Host, "h", ".", "RR to response to")
	flag.StringVar(&Host, "host", ".", "RR to response to")

	flag.StringVar(&Addr, "a", "0.0.0.0", "Address to listen on")
	flag.StringVar(&Addr, "address", "0.0.0.0", "Address to listen on")

	flag.Parse()
}

func main() {
	resolver := &Resolver{
		Port, Addr, Host,
	}
	log.Printf("Starting server at %v:%v, Host: %v\n", Addr, Port, Host)
	resolver.Serve()
}
