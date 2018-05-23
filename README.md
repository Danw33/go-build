# go-build: Multi-Project Build Utility

go-build is a multi-project, multi-branch build tool that can compile multiple branches of multiple projects to give per-branch and per-project artifacts.

This is ideal when working with many branches or repositories, as it allows easy publication or viewing of the compiled projects for testing and demonstration.

The tool can be used in combination with CI systems such as Travis and Jenkins to bundle artifacts of many projects for download and testing, and additionaly can be used with services such as GitHub Pages and GitLab Pages to publish live, user-testable development artifacts.

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
    - `scripts` - Array of script strings to execute (the build process)

### Run-time flags

The following flags can be passed to `go-build` at runtime:
  - `-v` - Verbose - Forces log level to `debug` (highest), Ignores log level set in config file.

## Prerequisites

`go-build` utilises the `git2go` bindings of `libgit2`, which require that libgit2 is
installed. In order to use SSH-based project urls, `libssh` is also required.


## License

The Multi-Project Build Tool is released under the MIT License

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
