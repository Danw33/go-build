BINARY=go-build
BUILDTIME=$(shell date +'%Y.%m.%d-%H:%M:%S')
VERSION=go-build-$(shell git describe --always --long --dirty)
GOFLAGS=-a -v
LDFLAGS=-ldflags="-X "main.Version=${VERSION}" -X "main.BuildTime=${BUILDTIME}" -s -w"
TRIMPATH=-trimpath="$(shell pwd)"
GCFLAGS=-gcflags=${TRIMPATH}
ASMFLAGS=-asmflags=${TRIMPATH}

default: build pack

dependencies:
	./build-dependencies.sh

build:
	go build ${GOFLAGS} ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BINARY}

build-static:
	go build ${GOFLAGS} --tags static  ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BINARY}

pack:
	upx -9 ${BINARY}

install:
	go install ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS}

install-static:
	go install --tags static ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS}

clean:
	go clean

.PHONY: all
