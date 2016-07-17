package main

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/miekg/dns"
)

const UDP = "udp"

// Resolver is a struct that keeps configured parameters for the DNS server.
// That would be Port (port to listen on), Addr (address to listen on) and Host
// (DNS Host that query should be queried). If the Host is configured as '.'
// (default) client can query ANY host. If the Host is non empty and doesn't
// match '.' client must query for A/AAAA record for that parameter.  Mind that
// Host value must end with a '.', for example "ip.example."
type Resolver struct {
	Port string
	Addr string
	Host string
}

// generateAnswerRecord build response record and writes back to
// dns.ResponseWriter object. The function will match IP version and queried record.
//
// If the source address is IPv4 and query type doesn't match rType A no record
// will be resolved and error will be returned.
// If the source address is IPv6
// and query type doesn't match rType AAAA, no record will be resolved and
// error will be returned.
//
// However, if the address is IPv4 and query type record is AAAA a IPv6 address
// will be returned (IPv4 in IPv6)
func generateAnswerRecord(host string, qType uint16, w dns.ResponseWriter,
	queryId uint16) (dns.RR, error) {

	log.Printf("[QueryID: %v] Source IP address: %v\n", queryId, w.RemoteAddr().String())
	remoteAddress, _ := net.ResolveUDPAddr(UDP, w.RemoteAddr().String())

	if remoteAddress.IP.To4() != nil && qType == dns.TypeA {
		return dns.NewRR(
			fmt.Sprintf("%s 0 IN A %s", host, remoteAddress.IP.String()))
	}
	if remoteAddress.IP.To16() != nil && qType == dns.TypeAAAA {
		return dns.NewRR(
			fmt.Sprintf("%s 0 IN AAAA %s", host, remoteAddress.IP.String()))
	}

	return nil, errors.New(fmt.Sprintf("Source address %v mismatches type %v\n",
		remoteAddress.IP.String(), dns.TypeToString[qType]))
}

// dnsHandler holds main logic of the application.
// It checks whether DNS packet is correct, fetches source IP address
// and builds appropriate DNS response message
func (resolver *Resolver) dnsHandler(w dns.ResponseWriter, r *dns.Msg) {
	queryId := r.MsgHdr.Id

	response := new(dns.Msg)
	response.SetReply(r)

	defer w.Close()
	defer w.WriteMsg(response)

	if len(r.Question) == 0 {
		response.Rcode = dns.RcodeFormatError
		return
	} else if len(r.Question) > 1 || r.Rcode != dns.OpcodeQuery {
		response.Rcode = dns.RcodeNotImplemented
		return
	}

	question := r.Question[0]

	if question.Qtype != dns.TypeA && question.Qtype != dns.TypeAAAA {
		return
	}

	host := question.Name
	log.Printf("[QueryID: %v] Got question for host: %v\n", queryId, host)

	if resolver.Host != "." && host != resolver.Host {
		log.Printf("[QueryID: %v] Host mismatch, got %v configured for %v\n",
			queryId, host, resolver.Host)
		return
	}

	answer, err := generateAnswerRecord(host, question.Qtype, w, queryId)
	if err != nil {
		log.Printf("[QueryID: %v] Error while generating answer record: %v\n", queryId, err)
	} else {
		response.Answer = append(response.Answer, answer)
	}
	return
}

// Serve runs DNS server based on provided (or default) parameters like address
// to listen on, port  or host
func (resolver *Resolver) Serve() {
	dns.HandleFunc(".", resolver.dnsHandler)
	server := &dns.Server{
		Addr: fmt.Sprintf("%s:%s", resolver.Addr, resolver.Port),
		Net:  UDP,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("%v\n", err)
	}
}
