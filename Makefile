BINARY=go-build
VERSION=go-build-dev-$(shell date +'%Y.%m.%d-%H:%M:%S')
LDFLAGS=-ldflags="-X main.Version=${VERSION} -s -w"
TRIMPATH=-trimpath="$(shell pwd)"
GCFLAGS=-gcflags=${TRIMPATH}
ASMFLAGS=-asmflags=${TRIMPATH}

default: build pack

dependencies:
	./build-dependencies.sh

build:
	go build -a -v ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BINARY}

build-static:
	go build -a -v --tags static  ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS} -o ${BINARY}

pack:
	upx -9 ${BINARY}

install:
	go install ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS}

install-static:
	go install --tags static ${LDFLAGS} ${GCFLAGS} ${ASMFLAGS}

clean:
	go clean

.PHONY: all
