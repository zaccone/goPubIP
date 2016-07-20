.PHONY: container

all: container

build:
	go build

container:
	go clean
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o goPubIP .
	install /etc/ssl/certs/ca-certificates.crt ca-certificates.crt
	docker build --rm -t=zaccone/gopubip .
	rm -rf ca-certificates.crt

clean:
	go clean
	rm -rf ca-certificates.crt
