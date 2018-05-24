#!/usr/bin/env bash

set -ex

pwd="$(pwd)"

export LDFLAGS="-L/usr/local/opt/openssl/lib $LDFLAGS"
export CPPFLAGS="-I/usr/local/opt/openssl/include $CPPFLAGS"
export CMAKE_PREFIX_PATH="/usr/lib;/usr/local/lib;/usr/lib/x86_64-linux-gnu;$CMAKE_PREFIX_PATH"
export PKG_CONFIG_PATH="/usr/local/opt/openssl/lib/pkgconfig;/usr/lib/pkgconfig;/usr/local/lib/pkgconfig;/usr/lib/x86_64-linux-gnu/pkgconfig;$PKG_CONFIG_PATH"

# Fetch go packages
go get -d github.com/op/go-logging
go get -d github.com/libgit2/git2go

rm -rf sources; mkdir sources ; cd sources
sources="$(pwd)"

# Fetch and build libssh from source
cd $sources
git clone git://git.libssh.org/projects/libssh.git --depth 1
mkdir build-libssh ; cd build-libssh
cmake ../libssh/
#make install

export CMAKE_PREFIX_PATH="$sources/build-libssh/lib;$CMAKE_PREFIX_PATH"
export PKG_CONFIG_PATH="$sources/build-libssh/lib/pkgconfig;$PKG_CONFIG_PATH"
export LDFLAGS="-L$sources/build-libssh/lib $LDFLAGS"
export CPPFLAGS="-I$sources/build-libssh/include $CPPFLAGS"

# Fetch and build libgit2 from source
cd $sources
git clone https://github.com/libgit2/libgit2.git --depth 1
mkdir build-libgit2 && cd build-libgit2
cmake ../libgit2/
cmake --build .
#make install

export CMAKE_PREFIX_PATH="$sources/build-libgit2/lib;$CMAKE_PREFIX_PATH"
export PKG_CONFIG_PATH="$sources/build-libgit2/lib/pkgconfig;$PKG_CONFIG_PATH"
export LDFLAGS="-L$sources/build-libgit2/lib $LDFLAGS"
export CPPFLAGS="-I$sources/build-libgit2/include $CPPFLAGS"

# Build git2go C bindings
cd "$GOPATH/src/github.com/libgit2/git2go"
git submodule update --init
make install-static
cd "$pwd"
