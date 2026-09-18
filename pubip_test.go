package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"testing"
	"time"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
)

// Embed the interface so unexpected use of other methods fails the test.
type recordingResponseWriter struct {
	dns.ResponseWriter
	remote  net.Addr
	message *dns.Msg
	closed  bool
}

func (w *recordingResponseWriter) RemoteAddr() net.Addr { return w.remote }
func (w *recordingResponseWriter) Conn() net.Conn       { return nil }
func (w *recordingResponseWriter) Write(data []byte) (int, error) {
	// WriteTo uses TCP framing for non-UDP test connections.
	if len(data) < 2 || int(binary.BigEndian.Uint16(data)) != len(data)-2 {
		return 0, fmt.Errorf("invalid DNS frame")
	}
	w.message = &dns.Msg{Data: append([]byte(nil), data[2:]...)}
	return len(data), w.message.Unpack()
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
			dnsutil.SetQuestion(request, "ip.example.", tt.qtype)
			request.CheckingDisabled = true
			request.AuthenticatedData = true
			request.Security = true
			request.UDPSize = 1232
			if err := request.Pack(); err != nil {
				t.Fatal(err)
			}
			writer := &recordingResponseWriter{remote: &net.UDPAddr{IP: net.ParseIP(tt.ip), Port: 12345}}
			resolver := &Resolver{Host: tt.host}
			resolver.dnsHandler(context.Background(), writer, request)
			if writer.closed || writer.message == nil {
				t.Fatal("handler must write a reply without closing the server socket")
			}
			reply := writer.message
			if reply.ID != request.ID || !reply.Response || reply.Rcode != dns.RcodeSuccess ||
				!reply.RecursionDesired || !reply.CheckingDisabled || reply.AuthenticatedData ||
				reply.Security || reply.UDPSize != 0 {
				t.Fatalf("unexpected reply header: %+v", reply.MsgHeader)
			}
			// Check the wire representation too, including IPv4-mapped AAAA answers.
			if err := reply.Pack(); err != nil {
				t.Fatal(err)
			}
			decoded := &dns.Msg{Data: reply.Data}
			if err := decoded.Unpack(); err != nil {
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
	for _, tt := range []struct {
		count int
		rcode uint16
	}{
		{0, dns.RcodeFormatError},
		{2, dns.RcodeNotImplemented},
	} {
		request := new(dns.Msg)
		for range tt.count {
			request.Question = append(request.Question, &dns.A{Hdr: dns.Header{Name: "ip.example.", Class: dns.ClassINET}})
		}
		// Exercise handler validation with an already decoded request.
		writer := new(recordingResponseWriter)
		resolver := &Resolver{Host: "."}
		resolver.dnsHandler(context.Background(), writer, request)
		if writer.message == nil || writer.message.Rcode != tt.rcode || len(writer.message.Answer) != 0 || writer.closed {
			t.Fatalf("%d questions: unexpected reply: %v", tt.count, writer.message)
		}
	}
}

// A response writer in v2 owns the shared UDP socket. Repeated queries catch
// accidentally closing that socket or sending an unchanged request buffer.
func TestDNSHandlerUDP(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	resolver := &Resolver{Host: "."}
	started := make(chan struct{})
	done := make(chan error, 1)
	server := &dns.Server{
		Net:               udp,
		PacketConn:        conn,
		Handler:           dns.HandlerFunc(resolver.dnsHandler),
		NotifyStartedFunc: func(context.Context) { close(started) },
	}
	go func() { done <- server.ListenAndServe() }()
	select {
	case <-started:
	case err := <-done:
		t.Fatalf("server failed to start: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("server stopped: %v", err)
			}
		case <-ctx.Done():
			t.Error("server did not stop")
		}
	})
	client, err := net.Dial("udp4", conn.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for _, qtype := range []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeTXT, dns.TypeA} {
		// CHAOS also reached the old root handler; v2's default mux is class-aware.
		for _, class := range []uint16{dns.ClassINET, dns.ClassCHAOS} {
			request := dnsutil.SetQuestion(new(dns.Msg), "ip.example.", qtype, class)
			if err := request.Pack(); err != nil {
				t.Fatal(err)
			}
			if err := client.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			if _, err := client.Write(request.Data); err != nil {
				t.Fatal(err)
			}
			buf := make([]byte, 4096)
			n, err := client.Read(buf)
			if err != nil {
				t.Fatal(err)
			}
			reply := &dns.Msg{Data: buf[:n]}
			if err := reply.Unpack(); err != nil {
				t.Fatal(err)
			}
			if !reply.Response || reply.ID != request.ID || reply.Rcode != dns.RcodeSuccess {
				t.Fatalf("unexpected reply: %v", reply)
			}
			want := ""
			switch qtype {
			case dns.TypeA:
				want = "ip.example.\t0\tIN\tA\t127.0.0.1"
			case dns.TypeAAAA:
				want = "ip.example.\t0\tIN\tAAAA\t::ffff:127.0.0.1"
			}
			if want == "" {
				if len(reply.Answer) != 0 {
					t.Fatalf("unexpected answers: %v", reply.Answer)
				}
			} else if len(reply.Answer) != 1 || reply.Answer[0].String() != want {
				t.Fatalf("answers = %v, want %s", reply.Answer, want)
			}
		}
	}
}
