# goPubIP
Ask me for an A or AAAA record and I will respond with your IP address

[![Go Report Card](https://goreportcard.com/badge/github.com/zaccone/goPubIP)](https://goreportcard.com/report/github.com/zaccone/goPubIP)
[![Build Status](https://travis-ci.org/zaccone/goPubIP.svg?branch=master)](https://travis-ci.org/zaccone/goPubIP)

## Installation
```
$ go get github.com/miekg/dns
$ go get github.com/zaccone/goPubIP
$ go install
```

## Usage

```
$ goPubIP -h
Usage of goPubIP:
  -a string
        Address to listen on, mind that IPv6 address must be in format [ip6address] (default "0.0.0.0")
  -address string
        Address to listen on, mind that IPv6 address must be in format [ip6address] (default "0.0.0.0")
  -h string
        RR to response to, host must end with a single dot ('.') (default ".")
  -host string
        RR to response to, host must end with a single dot ('.') (default ".")
  -p string
        Port to listen to (default "5300")
  -port string
        Port to listen to (default "5300")
```

## Running goPubIP

If you run it without any options goPubIP will by default listen on both IPv4 and IPv6 interfaces, on port 5300 and will respond to A
and AAAA queries for ANY host. If you want to limit proper responses to queries for a given host specify it as a {-h, --host} option.
Mind that host must end with a dot ('.').

How to run:

```
$ goPubIP -h ip.example.com.
```

Command above will run a server on 0.0.0.0:5300 and will respond to queries for ip.example.com {A,AAAA} record only.

## DNS response

By design any query other than A or AAAA will return in an empty response, that is, no IP address will be resolved.
If the caller's address is IPv6 and caller queries for A record an empty response will be returned.
If the caller's address is IPv4 and caller queries for either A record a response with callers source address will be returned.
If caller's address is IPv4 and caller queries for AAAA record goPubIP returns IPv6 address converted from callers IPv4 address
(i.e. if callers address is 127.0.0.1 the response will be ::ffff:127.0.0.1).


## Examples of client commands

```
$ dig +short -p 5300  ip.example.com @127.0.0.1
127.0.0.1
```

```
$ dig +short -p 5300 AAAA  ip.example.com @::1
::1
```

```
$ dig +short -p 5300 AAAA  ip.example.com @127.0.0.1
::ffff:127.0.0.1
```

## Invalid queries

Querying for an A record from a IPv6 source address will result in an empty response.

```
$ dig +short -p 5300 -6  A  ip.example.com @::1
```

Same, but with full response:

```
$ dig  -p 5300 -6  A  ip.example.com @::1

; <<>> DiG 9.9.5-9+deb8u6-Debian <<>> -p 5300 -6 A ip.example.com @::1
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 27926
;; flags: qr rd; QUERY: 1, ANSWER: 0, AUTHORITY: 0, ADDITIONAL: 0
;; WARNING: recursion requested but not available

;; QUESTION SECTION:
;ip.example.com.                        IN      A

;; Query time: 3 msec
;; SERVER: ::1#5300(::1)
;; WHEN: Mon Jul 18 15:22:23 CEST 2016
;; MSG SIZE  rcvd: 32
```

### Running goPubIP in Docker

To run basic resolver simply type in your command line:

```
$ docker run -d --name pubip -p 5300:5300/udp zaccone/gopubip:latest
```

You can specify goPubIP options (like -a, -h, -p) if you want and those will be reflected in the configuration, for instance:

```
$ docker run -d --name pubip -p 5300:5300/udp zaccone/gopubip:latest -h ip.example.com
$ docker logs pubip
  2016/07/18 22:16:52 Starting server at 0.0.0.0:5300, Query host: ip.example.com
```

Bear in mind that Docker by default won't respond to queries from IPv6 address.
