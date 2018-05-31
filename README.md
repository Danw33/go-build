# go-build: Multi-Project Build Utility [![Build Status](https://travis-ci.org/Danw33/go-build.svg?branch=master)](https://travis-ci.org/Danw33/go-build)

go-build is a multi-project, multi-branch build tool that can compile multiple branches of multiple projects to give per-branch and per-project artifacts.

This is ideal when working with many branches or repositories, as it allows easy publication or viewing of the compiled projects for testing and demonstration.

The tool can be used in combination with CI systems such as Travis and Jenkins to bundle artifacts of many projects for download and testing, and additionally can be used with services such as GitHub Pages and GitLab Pages to publish live, user-testable development artifacts.

## Configuration

`.build.json` - This file is the configuration loaded by the `go-build` utility.
It contains the definition of the repositories that will be built, including their
location (URL and Path), branches, scrips, and artifacts.
  - `home` - The "home" directory under which the utility will run (must be writable)
  - `async` - `true` to run builds in parallel, `false` to run in sequence.
  - `log` - Logger Configuration
    - `level` -  Log level, one of: `critical` (lowest), `error`, `warning`, `notice`, `info` (default), or `debug` (highest).
  - `projects` - Array of project definitions, made up of:
    - `url` - Git URL for the Project
    - `path` - Path to use when cloning, and Publishing artifacts (Slugified name)
    - `artifacts` - Path to extract built artifacts from
    - `branches` - Array of branch names to build or `['*']` for all remote branches.
    - `scripts` - Array of script strings to execute (the build process); May contain script variables (see below).

### Script Variables
The `scripts` section of the `go-build` project configuration may use the following variables which will be replaced before the script is executed:

 - `{{.Project}}` - The name (path) of the project.
 - `{{.Branch}}` - The branch under which the script is to run.
 - `{{.URL}}` - The clone url of the project.
 - `{{.Artifacts}}` - The path to the project's output artifacts.

Script variables are processed using go's [template](https://golang.org/pkg/text/template/) package, this gives a powerful set of Actions, Arguments, and Pipelines which can be combined with the above variables within a script.

### Run-time flags

The following flags can be passed to `go-build` at runtime:
  - `-v` - Verbose - Forces log level to `debug` (highest), Ignores log level set in config file.

## Prerequisites

`go-build` utilises the [git2go](https://github.com/libgit2/git2go) bindings of `libgit2`, which require that libgit2 is
installed. In order to use SSH-based project urls, `libssh2` and `libssl` are also required.

## Building

### Building on macOS (darwin)

Ensure `GOPATH` is set, then install prerequisites:

```bash
brew update
brew upgrade # Optional, but recommended
brew install git go upx # Up-to-date git, go compiler and upx packer (optional)
brew install openssl libssh2 libgit2 # libssl, libssh2, libgit2 (required)
```

Then, setup the environment to use the libraries installed via brew:

```bash
export PKG_CONFIG_PATH="/usr/local/lib/pkgconfig:/usr/local/lib"
export PKG_CONFIG_PATH="$PKG_CONFIG_PATH:/usr/local/opt/openssl/lib/pkgconfig"
export PKG_CONFIG_PATH="$PKG_CONFIG_PATH:/usr/local/opt/libssh2/lib/pkgconfig"
export OPENSSLDIR=/usr/local/opt/openssl
```
(Optional: Add the above to `~/.bashrc` or `~/.bash_profile` to persist between sessions)

Fetch the go packages:

```bash
go get -d github.com/op/go-logging
go get -d github.com/libgit2/git2go
```

And setup git2go's libgit2 submodule as per their documentation:

```bash
cd $GOPATH/src/github.com/libgit2/git2go
git submodule update --init
make install-static
```

Finally, build `go-build` using the supplied makefile:

```bash
make build
make pack # Optional: Pack the binary using UPX
```


### Building on Ubuntu (linux)

Ensure `GOPATH` is set and the prerequisite libraries are installed,
then setup the environment to use the libraries:

```bash
export PKG_CONFIG_PATH=/usr/lib/x86_64-linux-gnu/pkgconfig/
```
(Optional: Add the above to `~/.bashrc` or `~/.bash_profile` to persist between sessions)

Fetch the go packages:

```bash
go get -d github.com/op/go-logging
go get -d github.com/libgit2/git2go
```

And setup git2go's libgit2 submodule as per their documentation:

```bash
cd $GOPATH/src/github.com/libgit2/git2go
git submodule update --init
make install-static
```

Finally, build `go-build` using the supplied makefile:

```bash
make build
make pack # Optional: Pack the binary using UPX
```

### Building on Windows (win32)

Building `go-build` on Windows has not yet been attempted, if you have successfully compiled `libssh2`, `libgit2` and `go-build` to run natively under win32 please feel free to document it here and [open a PR](https://github.com/Danw33/go-build/pulls).

## License

[`go-build`](https://github.com/Danw33/go-build) is released under the MIT License

Copyright © 2017 - 2018 Daniel Wilson

Permission is hereby granted, free of charge, to any person
obtaining a copy of this software and associated documentation
files (the “Software”), to deal in the Software without
restriction, including without limitation the rights to use,
copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the
Software is furnished to do so, subject to the following
conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.
