/**
go-build - Mulit-Project Build Utility by @Danw33
MIT License

Copyright 2017 - 2018 Daniel Wilson <hello@danw.io>

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/op/go-logging"
	"github.com/getsentry/raven-go"
)

var (
	// Version string for the build, added by make (see the makefile)
	Version = "go-build-dev-unstable"

	// BuildTime of the build, added by make (see the makefile)
	BuildTime = "unspecified"
)

// Log is the logger interface
var Log = logging.MustGetLogger("example")

// format is the configured log string formatter
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

func main() {
	start := time.Now()

	// Setup logger, default to INFO level
	logBackend := logging.NewLogBackend(os.Stdout, "", 0)
	logBackendFormatted := logging.NewBackendFormatter(logBackend, format)
	logging.SetBackend(logBackendFormatted)
	logging.SetLevel(logging.INFO, "")

	// Check for verbose flag, if it's present, up the level to DEBUG
	verbose := false
	versionOnly := false
	for _, arg := range os.Args {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
			logging.SetLevel(logging.DEBUG, "")
		}

		if arg == "--version" {
			versionOnly = true
		}
	}

	if versionOnly {
		fmt.Println(Version)
		os.Exit(0)
	}

	Log.Info("\n",
		"go-build: Danw33's Multi-Project Build Utility\n",
		"          Copyright © Daniel Wilson, MIT License\n",
		"          https://github.com/Danw33/go-build\n",
		"          Version    : ", Version, "\n",
		"          Build Time : ", BuildTime, "\n",
		"          Host OS    : ", runtime.GOOS, "\n",
		"          Host Arch  : ", runtime.GOARCH, "\n")

	Log.Debug("Finding working directory...")

	cwd, err := os.Getwd()
	if err != nil {
		Log.Critical(err)
		panic(err)
	}
	pwd = cwd

	Log.Debug("Reading configuration file...")
	configFile := ".build.json"
	cfgByte, err := ioutil.ReadFile(configFile)
	if err != nil {
		Log.Critical(err)
		panic(err)
	}
	cfg := string(cfgByte)
	config := parseConfig(cfg)

	// Prepare Raven
	if config.Metrics != false {
		// Metrics are enabled (well... not disabled)
		if config.RavenDSN != "" {
			// Use the custom DSN from the config
			Log.Info("Metrics are enabled using a custom DSN")
			raven.SetDSN( config.RavenDSN )
		} else {
			// Use the go-build master DSN
			Log.Info("Metrics are enabled using the default DSN")
			raven.SetDSN("https://b5e7240f7ba34112a46080e6064de695:3c281fda39154166abe538d1c50f6f7f@sentry.io/1265577")
		}
	} else {
		Log.Info("Metrics have been disabled")
	}

	var tags map[string]string
	tags = make(map[string]string)

	tags["version"] = Version
	tags["buildtime"] = BuildTime
	tags["goos"] = runtime.GOOS
	tags["goarch"] = runtime.GOARCH

	raven.SetRelease(Version)
	raven.SetTagsContext(tags)

	// Adjust the log level again, this time from the configuration file, but only if verbose isn't passed
	if verbose == false {
		level, err := logging.LogLevel(config.Log.Level)
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Critical(err)
		}
		logging.SetLevel(level, "")
	}

	// Check the configured home path
	if config.Home == "" || config.Home == "./" {
		Log.Debugf("config.Home has been left blank or configured relative, the current working directory will be used.")
		config.Home = pwd
	}

	Log.Infof("Configuration Loaded.")

	Log.Infof("Loading Plugins...")
	loadPlugins(config, []byte(cfg))
	runPostLoadPlugins(&Version, &BuildTime)

	cloneOpts := configureCloneOpts()

	Log.Debug("Starting Project Processor...")

	runPreProcessProjects(&pwd, &config.Home, &config.Async)
	processProjects(config, cloneOpts)
	runPostProcessProjects(&pwd, &config.Home, &config.Async)

	Log.Infof("All projects completed in: %s", time.Since(start))
}
