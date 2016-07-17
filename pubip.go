package main

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/miekg/dns"
)

type Context struct {
	IP   string
	Port string
	Host string
}

func generateAnswerRecord(host string, w dns.ResponseWriter) (dns.RR, error) {
	remoteAddress := w.RemoteAddr().String()
	if net.ParseIP(remoteAddress).To16() != nil {
		return dns.NewRR(fmt.Sprintf("%s A %s", host, remoteAddress))
	}

	if net.ParseIP(remoteAddress).To4() != nil {
		return dns.NewRR(fmt.Sprintf("%s AAAA %s", host, remoteAddress))
	}

	return nil, errors.New("Invalid IP address")

}

// dnsHandler holds main logic of the application.
// It checks whether DNS packet is correct, fetches source IP address
// and builds appropriate DNS response message
func dnsHandler(w dns.ResponseWriter, r *dns.Msg) {
	defer w.Close()
	const host = "ip.octogan.net"
	answer, _ := generateAnswerRecord(host, w)

	response := new(dns.Msg)
	response.SetReply(r)
	response.Answer = append(response.Answer, answer)
	w.WriteMsg(response)
}

func Serve(context Context) {
	dns.HandleFunc(".", dnsHandler)
	server := &dns.Server{
		Addr: ":53",
		Net:  "udp",
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	context := Context{
		Host: "ip.octogan.net",
	}
	Serve(context)
}
