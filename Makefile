.PHONY: all install build deps container-bin container clean

all: deps build

install:
	go install

build:
	go build

deps:
	go mod download

container-bin:
	CGO_ENABLED=0 GOOS=linux go build -o goPubIP .

container: deps container-bin
	install /etc/ssl/certs/ca-certificates.crt ca-certificates.crt
	docker build --rm -t=zaccone/gopubip .
	rm -rf ca-certificates.crt

clean:
	go clean
	rm -rf ca-certificates.crt
