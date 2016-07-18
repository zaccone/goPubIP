package main

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/miekg/dns"
)

const UDP = "udp"

type Resolver struct {
	Port string
	Addr string
	Host string
}

func qTypeToString(qType uint16) string {
	if qType == dns.TypeA {
		return "A"
	} else if qType == dns.TypeAAAA {
		return "AAAA"
	}

	return ""
}

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
		remoteAddress.IP.String(), qTypeToString(qType)))
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
		// return some err
		return
	}

	question := r.Question[0]

	if question.Qtype != dns.TypeA && question.Qtype != dns.TypeAAAA {
		//w.WriteMsg(response)
		return
	}

	host := question.Name
	log.Printf("[QueryID: %v] Got question for host: %v\n", queryId, host)

	if resolver.Host != "." && host != resolver.Host {
		log.Printf("[QueryID: %v] Host mismatch, got %v configured for %v\n",
			queryId, host, resolver.Host)
		//	w.WriteMsg(response)
		return
	}

	answer, err := generateAnswerRecord(host, question.Qtype, w, queryId)
	if err != nil {
		log.Printf("[QueryID: %v] Error while generating answer record: %v\n", queryId, err)
	} else {
		response.Answer = append(response.Answer, answer)
	}
	//	w.WriteMsg(response)
}

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
