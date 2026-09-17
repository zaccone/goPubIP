package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/rdata"
)

const udp = "udp"

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
	queryID uint16) (dns.RR, error) {

	log.Printf("[QueryID: %v] Source IP address: %v\n", queryID, w.RemoteAddr().String())
	remoteAddress, err := netip.ParseAddrPort(w.RemoteAddr().String())
	if err != nil {
		return nil, err
	}
	ip := remoteAddress.Addr().Unmap()

	if ip.Is4() && qType == dns.TypeA {
		return &dns.A{
			Hdr: dns.Header{Name: host, Class: dns.ClassINET},
			A:   rdata.A{Addr: ip},
		}, nil
	}
	if ip.IsValid() && qType == dns.TypeAAAA {
		// As16 preserves the documented IPv4-mapped IPv6 answer.
		return &dns.AAAA{
			Hdr:  dns.Header{Name: host, Class: dns.ClassINET},
			AAAA: rdata.AAAA{Addr: netip.AddrFrom16(ip.As16())},
		}, nil
	}

	return nil, fmt.Errorf("Source address %v mismatches type %v\n",
		ip.String(), dns.TypeToString[qType])
}

// dnsHandler holds main logic of the application.
// It checks whether DNS packet is correct, fetches source IP address
// and builds appropriate DNS response message
func (resolver *Resolver) dnsHandler(_ context.Context, w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Data) > 0 {
		if err := r.Unpack(); err != nil {
			log.Printf("Error unpacking query: %v", err)
			return
		}
	}
	queryID := r.ID
	requestHeader := r.MsgHeader
	questionCount := len(r.Question)

	// Reuse the incoming message so WriteTo returns its buffer to the server pool.
	response := r
	response.Reset()
	if questionCount > 0 {
		response.Question = response.Question[:1]
	}
	response.MsgHeader = dns.MsgHeader{ID: queryID, Opcode: requestHeader.Opcode, Response: true}
	if requestHeader.Opcode == dns.OpcodeQuery {
		response.RecursionDesired = requestHeader.RecursionDesired
		response.CheckingDisabled = requestHeader.CheckingDisabled
	}
	// The server owns the UDP socket; closing the writer would stop the listener.
	defer func() {
		if err := response.Pack(); err != nil {
			log.Printf("[QueryID: %v] Error packing response: %v", queryID, err)
			return
		}
		if _, err := response.WriteTo(w); err != nil {
			log.Printf("[QueryID: %v] Error writing response: %v", queryID, err)
		}
	}()

	if questionCount == 0 {
		response.Rcode = dns.RcodeFormatError
		return
	} else if questionCount > 1 || requestHeader.Rcode != dns.OpcodeQuery {
		response.Rcode = dns.RcodeNotImplemented
		return
	}

	question := r.Question[0]

	if dns.RRToType(question) != dns.TypeA && dns.RRToType(question) != dns.TypeAAAA {
		return
	}

	host := question.Header().Name
	log.Printf("[QueryID: %v] Got question for host: %v\n", queryID, host)

	if resolver.Host != "." && host != resolver.Host {
		log.Printf("[QueryID: %v] Host mismatch, got %v configured for %v\n",
			queryID, host, resolver.Host)
		return
	}

	answer, err := generateAnswerRecord(host, dns.RRToType(question), w, queryID)
	if err != nil {
		log.Printf("[QueryID: %v] Error while generating answer record: %v\n", queryID, err)
	} else {
		response.Answer = append(response.Answer, answer)
	}
	return
}

// Serve runs DNS server based on provided (or default) parameters like address
// to listen on, port  or host
func (resolver *Resolver) Serve() {
	server := &dns.Server{
		Addr:    fmt.Sprintf("%s:%s", resolver.Addr, resolver.Port),
		Net:     udp,
		Handler: dns.HandlerFunc(resolver.dnsHandler),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("%v\n", err)
	}
}
