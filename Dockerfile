FROM golang:alpine
RUN apk update
RUN apk add --no-cache gcc musl-dev libc-dev g++ alpine-sdk bash git go upx make cmake python3 openssl-dev curl libcurl curl-dev libgcrypt libgcrypt-dev libssh libssh-dev libssh2 libssh2-dev
COPY . /go-build
WORKDIR /go-build
RUN . ./build-dependencies.sh && make build