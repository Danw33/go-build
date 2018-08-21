GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
BINARY=go-build-${GOOS}-${GOARCH}
BUILDTIME=$(shell date +'%Y.%m.%d-%H:%M:%S')
VERSION=go-build-$(shell git describe --always --long --dirty)
GOFLAGS=-v
LDFLAGS=-ldflags="-X "main.Version=${VERSION}" -X "main.BuildTime=${BUILDTIME}" -s -w"
TRIMPATH=-trimpath="$(shell pwd)"
GCFLAGS=-gcflags=${TRIMPATH}
ASMFLAGS=-asmflags=${TRIMPATH}

default: build pack

dependencies:
	./build-dependencies.sh

build:
	go build ${GOFLAGS} ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BINARY} ./src

build-static:
	go build ${GOFLAGS} --tags static  ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BINARY} ./src

build-debug:
	GOCACHE=off go build -x -tags nopkcs11 -ldflags='-X "main.Version=${VERSION}-dbg" -X "main.BuildTime=${BUILDTIME}"' -gcflags='all=-N -l -dwarflocationlists=true' -o ${BINARY}-dbg ./src

build-plugins:
	go build ${GOFLAGS} -buildmode=plugin ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} ./plugins/example/*.go
	go build ${GOFLAGS} -buildmode=plugin ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} ./plugins/all-branches/*.go
	go build ${GOFLAGS} -buildmode=plugin ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} ./plugins/index-generator/*.go

build-docker:
	docker build -t ${VERSION} .

pack:
	upx -9 ${BINARY}

install:
	go install ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} ./src

install-static:
	go install --tags static ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} ./src

clean:
	go clean

.PHONY: all
