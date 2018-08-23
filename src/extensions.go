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

	"github.com/getsentry/raven-go"
)

// loadedPlugins contains the raw plugins loaded from the filesystem
var loadedPlugins []*plugin.Plugin

// buildPlugins contains the actual BuildPlugin symbols for each loaded plugin
var buildPlugins []BuildPlugin

// loadPlugins is responsible for reading, testing, and initialising plugins that
// have been defined in the configuration file.
func loadPlugins(config *Configuration, rawCfg []byte) {

	// See if the config defines any plugins
	if len(config.Plugins) == 0 {
		Log.Info("No plugins configured, bypassing plugin loader.")
		return
	}

	// Load in plugin files
	for _, pFile := range config.Plugins {
		raven.CapturePanic(func() {
			if plug, err := plugin.Open(pFile); err == nil {
				loadedPlugins = append(loadedPlugins, plug)
			} else {
				Log.Criticalf("Failed to load plugin \"%s\"", pFile)
				Log.Critical(err)
			}
		}, nil)
	}

	// See if we loaded any plugins from the disk
	if len(loadedPlugins) == 0 {
		Log.Info("No configured plugins could be loaded.")
		return
	}

	// For each plugin we loaded, Locate BuildPlugin symbol and check it
	for _, p := range loadedPlugins {
		// Lookup the symbol
		sym, err := p.Lookup("BuildPlugin")
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf("Plugin exports no BuildPlugin symbol: %v", err)
			continue
		}

		// Check it's compatible with our definition of the BuildPlugin interface
		bp, ok := sym.(BuildPlugin)
		if !ok {
			Log.Errorf("Build Plugin is not an BuildPlugin interface type")
			continue
		}

		// Call the pluginInit for the current plugin
		initErr := bp.PluginInit(rawCfg)
		if initErr != nil {
			raven.CaptureError(initErr, nil)
			Log.Errorf("Plugin loaded but failed to initialise: %v", initErr)
			continue
		}

		// Add it to the array of initialised plugins
		buildPlugins = append(buildPlugins, bp)
	}

	// Log plugin status now loading is completed
	Log.Debugf("Plugin Loader: %d found in config, %d loaded from filesystem, %d compatible and initialised.", len(config.Plugins), len(loadedPlugins), len(buildPlugins))
	Log.Infof("Initialised %d plugins successfully", len(buildPlugins))
}

// postLoadPlugins (1) is the first fully-loaded hook, after all plugins are loaded
func runPostLoadPlugins(version *string, buildTime *string) {
	for _, lp := range buildPlugins {
		raven.CapturePanic(func() {
			lp.PostLoadPlugins(version, buildTime)
		}, nil)
	}
}

// preProcessProjects (2) is run before processing all projects
func runPreProcessProjects(workingDir *string, homeDir *string, async *bool) {
	for _, lp := range buildPlugins {
		raven.CapturePanic(func() {
			lp.PreProcessProjects(workingDir, homeDir, async)
		}, nil)
	}
}

// postProcessProjects (9) is run after processing all projects
func runPostProcessProjects(workingDir *string, homeDir *string, async *bool) {
	for _, lp := range buildPlugins {
		raven.CapturePanic(func() {
			lp.PostProcessProjects(workingDir, homeDir, async)
		}, nil)
	}
}

// preProcessProject (3) is run before processing an individual project
func runPreProcessProject(url *string, path *string, artifacts *string, branches *[]string, scripts *[]string) {
	for _, lp := range buildPlugins {
		raven.CapturePanic(func() {
			lp.PreProcessProject(url, path, artifacts, branches, scripts)
		}, nil)
	}
}

// postProcessProject (8) is run after processing an individual project
func runPostProcessProject(url *string, path *string, artifacts *string, branches *[]string, scripts *[]string) {
	for _, lp := range buildPlugins {
		raven.CapturePanic(func() {
			lp.PostProcessProject(url, path, artifacts, branches, scripts)
		}, nil)
	}
}

// preProcessBranch (4) is run before processing a branch within a project
func runPreProcessBranch(projectDir *string, branchName *string, workDirDesc *string) {
	for _, lp := range buildPlugins {
		raven.CapturePanic(func() {
			lp.PreProcessBranch(projectDir, branchName, workDirDesc)
		}, nil)
	}
}

// postProcessBranch (7) is run after processing a branch within a project
func runPostProcessBranch(projectDir *string, branchName *string, workDirDesc *string) {
	for _, lp := range buildPlugins {
		raven.CapturePanic(func() {
			lp.PostProcessBranch(projectDir, branchName, workDirDesc)
		}, nil)
	}
}

// preProcessArtifacts (5) is run before processing the build artifacts of a branch
func runPreProcessArtifacts(artifactPath *string, projectPath *string, branchName *string) {
	for _, lp := range buildPlugins {
		raven.CapturePanic(func() {
			lp.PreProcessArtifacts(artifactPath, projectPath, branchName)
		}, nil)
	}
}

// postProcessArtifacts (6) is run after processing the build artifacts of a branch
func runPostProcessArtifacts(artifactPath *string, projectPath *string, branchName *string) {
	for _, lp := range buildPlugins {
		raven.CapturePanic(func() {
			lp.PostProcessArtifacts(artifactPath, projectPath, branchName)
		}, nil)
	}
}
