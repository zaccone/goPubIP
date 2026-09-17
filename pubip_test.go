package main

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

// Embed the interface so unexpected use of other methods fails the test.
type recordingResponseWriter struct {
	dns.ResponseWriter
	remote  net.Addr
	message *dns.Msg
	closed  bool
}

func (w *recordingResponseWriter) RemoteAddr() net.Addr { return w.remote }
func (w *recordingResponseWriter) WriteMsg(message *dns.Msg) error {
	w.message = message.Copy()
	return nil
}
func (w *recordingResponseWriter) Close() error {
	w.closed = true
	return nil
}

func TestDNSHandler(t *testing.T) {
	tests := []struct {
		name  string
		ip    string
		qtype uint16
		host  string
		want  string
	}{
		{"IPv4 A", "192.0.2.1", dns.TypeA, ".", "ip.example.\t0\tIN\tA\t192.0.2.1"},
		{"IPv4 AAAA", "192.0.2.1", dns.TypeAAAA, ".", "ip.example.\t0\tIN\tAAAA\t::ffff:192.0.2.1"},
		{"IPv6 AAAA", "2001:db8::1", dns.TypeAAAA, ".", "ip.example.\t0\tIN\tAAAA\t2001:db8::1"},
		{"IPv6 A", "2001:db8::1", dns.TypeA, ".", ""},
		{"unsupported type", "192.0.2.1", dns.TypeTXT, ".", ""},
		{"matching host", "192.0.2.1", dns.TypeA, "ip.example.", "ip.example.\t0\tIN\tA\t192.0.2.1"},
		{"different host", "192.0.2.1", dns.TypeA, "other.example.", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := new(dns.Msg)
			request.SetQuestion("ip.example.", tt.qtype)
			writer := &recordingResponseWriter{remote: &net.UDPAddr{IP: net.ParseIP(tt.ip), Port: 12345}}
			resolver := &Resolver{Host: tt.host}
			resolver.dnsHandler(writer, request)
			if !writer.closed || writer.message == nil {
				t.Fatal("handler must write a reply and close the writer")
			}
			reply := writer.message
			if reply.Id != request.Id || !reply.Response || reply.Rcode != dns.RcodeSuccess {
				t.Fatalf("unexpected reply header: %+v", reply.MsgHdr)
			}
			// Check the wire representation too, including IPv4-mapped AAAA answers.
			wire, err := reply.Pack()
			if err != nil {
				t.Fatal(err)
			}
			decoded := new(dns.Msg)
			if err := decoded.Unpack(wire); err != nil {
				t.Fatal(err)
			}
			if tt.want == "" {
				if len(decoded.Answer) != 0 {
					t.Fatalf("unexpected answers: %v", decoded.Answer)
				}
			} else if len(decoded.Answer) != 1 || decoded.Answer[0].String() != tt.want {
				t.Fatalf("answers = %v, want %s", decoded.Answer, tt.want)
			}
		})
	}
}

func TestDNSHandlerQuestionCount(t *testing.T) {
	for _, tt := range []struct{ count, rcode int }{
		{0, dns.RcodeFormatError},
		{2, dns.RcodeNotImplemented},
	} {
		request := new(dns.Msg)
		for range tt.count {
			request.Question = append(request.Question, dns.Question{Name: "ip.example.", Qtype: dns.TypeA, Qclass: dns.ClassINET})
		}
		writer := new(recordingResponseWriter)
		resolver := &Resolver{Host: "."}
		resolver.dnsHandler(writer, request)
		if writer.message == nil || writer.message.Rcode != tt.rcode || len(writer.message.Answer) != 0 || !writer.closed {
			t.Fatalf("%d questions: unexpected reply: %v", tt.count, writer.message)
		}
	}
}
