package main

import (
	"github.com/op/go-logging"
)

type pluginName struct{}

var log *logging.Logger
var config Configuration

// pluginInit (0) is the Plugin Initialiser, called on load of plugin file
func (t pluginName) pluginInit(logger *logging.Logger, configuration Configuration, coreVersion string) {
	log = logger
	config = configuration
	log.Debugf("pluginName: loaded and initialised, detected core version as %s", coreVersion)
}

// postLoadPlugins (1) is the first fully-loaded hook, after all plugins are loaded
func (t pluginName) postLoadPlugins() {}

// preProcessProjects (2) is run before processing all projects
func (t pluginName) preProcessProjects() {}

// postProcessProjects (9) is run after processing all projects
func (t pluginName) postProcessProjects() {}

// preProcessProject (3) is run before processing an individual project
func (t pluginName) preProcessProject() {}

// postProcessProject (8) is run after processing an individual project
func (t pluginName) postProcessProject() {}

// preProcessBranch (4) is run before processing a branch within a project
func (t pluginName) preProcessBranch() {}

// postProcessBranch (7) is run after processing a branch within a project
func (t pluginName) postProcessBranch() {}

// preProcessArtifacts (5) is run before processing the build artifacts of a branch
func (t pluginName) preProcessArtifacts() {}

// postProcessArtifacts (6) is run after processing the build artifacts of a branch
func (t pluginName) postProcessArtifacts() {}

// Export BuildPlugin symbol
var BuildPlugin pluginName
