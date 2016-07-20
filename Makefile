.PHONY: container

all: deps build

build:
	go build

deps:
	go get -u github.com/miekg/dns

container-bin:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o goPubIP .

container: deps container-bin
	install /etc/ssl/certs/ca-certificates.crt ca-certificates.crt
	docker build --rm -t=zaccone/gopubip .
	rm -rf ca-certificates.crt

clean:
	go clean
	rm -rf ca-certificates.crt
