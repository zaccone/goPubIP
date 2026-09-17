# goPubIP

A small UDP DNS server that replies to A and AAAA queries with the client's
source IP address. Use it to discover the address the server sees for a client.
It does not perform recursive DNS lookups.

[![Go CI](https://github.com/zaccone/goPubIP/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/zaccone/goPubIP/actions/workflows/go.yml)

## Installation

Requires Go 1.27 or newer.

```sh
go install github.com/zaccone/goPubIP@latest
```

The binary is installed in `GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is
unset. Add that directory to your `PATH`.

To build from source with the pinned module dependencies:

```sh
git clone https://github.com/zaccone/goPubIP.git
cd goPubIP
make all
./goPubIP
```

Run `make install` to install your local checkout, or `make clean` to remove
local build output.

## Quick start

Start a server on port 5300 that answers queries for `ip.example.com.`:

```sh
goPubIP -host ip.example.com.
```

In another terminal, query it with `dig`:

```sh
dig +short @127.0.0.1 -p 5300 ip.example.com. A
# 127.0.0.1

dig +short @127.0.0.1 -p 5300 ip.example.com. AAAA
# ::ffff:127.0.0.1
```

Replace `127.0.0.1` after `@` with your server's address to query it remotely.
The reply contains the source address seen by the server, which may be a NAT
or DNS forwarder's address if the query passes through one.

## Options

| Short flag | Long flag | Default | Description |
| --- | --- | --- | --- |
| `-a` | `-address` | `0.0.0.0` | Address to listen on. Enclose IPv6 addresses in brackets. |
| `-p` | `-port` | `5300` | UDP port to listen on. |
| `-h` | `-host` | `.` | Query name to answer. `.` accepts any name. |

A configured host must end with a dot, for example `ip.example.com.`. Host
matching is exact, including letter case. Queries for other names receive an
empty answer.

`-h` requires a host value; it is not a help flag. Display usage with:

```sh
goPubIP -help
```

With no options, the server listens on `0.0.0.0:5300` and accepts any query name.
IPv6 availability on this wildcard listener depends on the operating system.
To listen explicitly on IPv6 loopback:

```sh
goPubIP -address '[::1]' -port 5300 -host ip.example.com.
```

Query that listener from another terminal:

```sh
dig +short @::1 -p 5300 ip.example.com. AAAA
# ::1
```

## Response behavior

| Client source address | Query type | Answer |
| --- | --- | --- |
| IPv4 | A | Client's IPv4 address |
| IPv4 | AAAA | IPv4-mapped IPv6 address, such as `::ffff:192.0.2.1` |
| IPv6 | AAAA | Client's IPv6 address |
| IPv6 | A | Empty answer |
| Either | Any other type | Empty answer |

Answers have a TTL of zero. Empty answers for unsupported types or host/address
mismatches use the DNS `NOERROR` response code; they are not `NXDOMAIN` replies.

For example, an A query to the IPv6 loopback listener returns no answer records:

```sh
dig @::1 -p 5300 ip.example.com. A
```

## Docker

Build the image from your checkout on Linux with Go, Make, Docker, and a CA
certificate bundle at `/etc/ssl/certs/ca-certificates.crt`:

```sh
make container
```

The existing build target produces a static Linux binary for the build machine's
architecture and packages it in a `scratch` image named `zaccone/gopubip`.

Run the locally built image and publish its UDP port:

```sh
docker run -d --name pubip -p 5300:5300/udp zaccone/gopubip -host ip.example.com.
docker logs pubip
```

Application flags go after the image name. To answer queries for any host, omit
`-host ip.example.com.`. IPv6 access depends on your Docker host and network
configuration.

Stop and remove the container when finished:

```sh
docker stop pubip
docker rm pubip
```

## Development

```sh
go mod verify
go vet ./...
go test -race ./...
make build
```

`make deps` downloads the versions pinned in `go.mod` and `go.sum`; it does not
upgrade them.

## License

[MIT](LICENSE)
