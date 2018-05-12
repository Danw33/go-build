BINARY=go-build
VERSION=go-build-dev-$(shell date +'%Y.%m.%d-%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

build:
	go build --tags static  ${LDFLAGS} -o ${BINARY}

pack:
	upx -9 ${BINARY}

install:
	go install --tags static ${LDFLAGS}

clean:
	go clean

.PHONY: build pack install clean
