package main

import (
	"flag"
	"log"
)

var (
	port string
	addr string
	host string
)

func init() {
	PortStr := "Port to listen to"
	flag.StringVar(&port, "p", "5300", PortStr)
	flag.StringVar(&port, "port", "5300", PortStr)

	HostStr := "RR to response to, host must end with a single dot ('.')"
	flag.StringVar(&host, "h", ".", HostStr)
	flag.StringVar(&host, "host", ".", HostStr)

	AddrStr := "Address to listen on, mind that IPv6 address must be in format [ip6address]"
	flag.StringVar(&addr, "a", "0.0.0.0", AddrStr)
	flag.StringVar(&addr, "address", "0.0.0.0", AddrStr)

	flag.Parse()
}

func main() {
	resolver := &Resolver{
		port, addr, host,
	}
	log.Printf("Starting server at %v:%v, Query host: %v\n", addr, port, host)
	resolver.Serve()
}
