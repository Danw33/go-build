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

// extension - Plugins
package main

import (
	"plugin"
	"github.com/op/go-logging"
)

// BuildPlugin is an interface for go-build plugins
type BuildPlugin interface {
	// pluginInit 0. Plugin Initialiser, called on load of plugin file
	pluginInit(*logging.Logger, configuration, string) error

	// postLoadPlugins 1. First hook, after plugins are loaded
	postLoadPlugins([]*plugin.Plugin, configuration)

	// preProcessProjects 2. Before processing all projects
	preProcessProjects()

	// postProcessProjects 9. After processing all projects
	postProcessProjects()

	// preProcessProject 3. Before processing an individual project
	preProcessProject()

	// postProcessProject 8. After processing an individual project
	postProcessProject()

	// preProcessBranch 4. Before processing a branch within a project
	preProcessBranch()

	// postProcessBranch 7. After processing a branch within a project
	postProcessBranch()

	// preProcessArtifacts 5. Before processing the build artifacts of a branch
	preProcessArtifacts()

	// postProcessArtifacts 6. After processing the build artifacts of a branch
	postProcessArtifacts()
}

var loadedPlugins []*plugin.Plugin
var buildPlugins []BuildPlugin

func loadPlugins(config *configuration) {
	// Load in plugin files
	for _, pFile := range config.Plugins {
		if plug, err := plugin.Open(pFile); err == nil {
			loadedPlugins = append(loadedPlugins, plug)
		} else {
			log.Criticalf("Failed to load plugin \"%s\"", pFile)
			log.Critical(err)
		}
	}

	// Locate buildPlugin symbols and test
	for _, p := range loadedPlugins {
		symPlug, err := p.Lookup("BuildPlugin")
		if err != nil {
			log.Errorf("Plugin exports no BuildPlugin symbol: %v", err)
			continue
		}
		bp, ok := symPlug.(BuildPlugin)
		if !ok {
			log.Errorf("Build Plugin is not an BuildPlugin interface type")
			continue
		}
		buildPlugins = append(buildPlugins, bp)
	}
}
