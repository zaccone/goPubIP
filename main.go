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
	PortStr := "Port to listen to"
	flag.StringVar(&Port, "p", "5300", PortStr)
	flag.StringVar(&Port, "port", "5300", PortStr)

	HostStr := "RR to response to, host must end with a single dot ('.')"
	flag.StringVar(&Host, "h", ".", HostStr)
	flag.StringVar(&Host, "host", ".", HostStr)

	AddrStr := "Address to listen on, mind that IPv6 address must be in format [ip6address]"
	flag.StringVar(&Addr, "a", "0.0.0.0", AddrStr)
	flag.StringVar(&Addr, "address", "0.0.0.0", AddrStr)

	flag.Parse()
}

func main() {
	resolver := &Resolver{
		Port, Addr, Host,
	}
	log.Printf("Starting server at %v:%v, Query host: %v\n", Addr, Port, Host)
	resolver.Serve()
}
