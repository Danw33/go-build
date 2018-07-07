// go-build-plugin-example
package main

import (
	"fmt"
)

type BuildPluginImpl struct{}

// pluginInit (0) is the Plugin Initialiser, called on load of plugin file
func (b BuildPluginImpl) PluginInit() error {
	fmt.Println("Yo, EX-to-the-A to-the-M to-the-PLE.")
	fmt.Println("Yeah that's right; I'm the example plugin.")
	return nil
}

// postLoadPlugins (1) is the first fully-loaded hook, after all plugins are loaded
func (b BuildPluginImpl) PostLoadPlugins() {
	fmt.Println("Example Plugin: PostLoadPlugins - All plugins have loaded")
}

// preProcessProjects (2) is run before processing all projects
func (b BuildPluginImpl) PreProcessProjects() {
	fmt.Println("Example Plugin: PreProcessProjects - Just about to start processing the projects")
}

// postProcessProjects (9) is run after processing all projects
func (b BuildPluginImpl) PostProcessProjects() {
	fmt.Println("Example Plugin: PreProcessProjects - Just finished processing the projects")
}

// preProcessProject (3) is run before processing an individual project
func (b BuildPluginImpl) PreProcessProject() {
	fmt.Println("Example Plugin: PreProcessProjects - Just about to start processing an individual project")
}

// postProcessProject (8) is run after processing an individual project
func (b BuildPluginImpl) PostProcessProject() {
	fmt.Println("Example Plugin: PostProcessProject - Just finished processing an individual project")
}

// preProcessBranch (4) is run before processing a branch within a project
func (b BuildPluginImpl) PreProcessBranch() {
	fmt.Println("Example Plugin: PreProcessBranch - Just about to start processing a branch of an individual project")
}

// postProcessBranch (7) is run after processing a branch within a project
func (b BuildPluginImpl) PostProcessBranch() {
	fmt.Println("Example Plugin: PostProcessBranch - Just finished processing a branch of an individual project")
}

// preProcessArtifacts (5) is run before processing the build artifacts of a branch
func (b BuildPluginImpl) PreProcessArtifacts() {
	fmt.Println("Example Plugin: PreProcessArtifacts - Just about to start processing artifacts for a branch of an individual project")
}

// postProcessArtifacts (6) is run after processing the build artifacts of a branch
func (b BuildPluginImpl) PostProcessArtifacts() {
	fmt.Println("Example Plugin: PreProcessArtifacts - Just finished processing artifacts for a branch of an individual project")
}

var BuildPlugin BuildPluginImpl
