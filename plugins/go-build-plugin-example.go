// go-build-plugin-example
package main

import (
	"fmt"
)

type BuildPluginImpl struct{}

// pluginInit (0) is the Plugin Initialiser, called on load of plugin file
func (b BuildPluginImpl) PluginInit(rawConfig []byte) error {
	fmt.Println("Yo, EX-to-the-A to-the-M to-the-PLE.")
	fmt.Println("Yeah that's right; I'm the example plugin.")
	fmt.Println("I've been given the following byte array to represent the raw configuration:")
	fmt.Println(rawConfig)
	return nil
}

// postLoadPlugins (1) is the first fully-loaded hook, after all plugins are loaded
func (b BuildPluginImpl) PostLoadPlugins(version *string, buildTime *string) {
	fmt.Println("Example Plugin: PostLoadPlugins - All plugins have loaded, we know the core was built at", *buildTime, "and is version", *version)
}

// preProcessProjects (2) is run before processing all projects
func (b BuildPluginImpl) PreProcessProjects(workingDir *string, homeDir *string, async *bool) {
	fmt.Println("Example Plugin: PreProcessProjects - Just about to start processing the projects, I know that we're in", *workingDir, " and the configuration wants us in", *homeDir)
}

// postProcessProjects (9) is run after processing all projects
func (b BuildPluginImpl) PostProcessProjects(workingDir *string, homeDir *string, async *bool) {
	fmt.Println("Example Plugin: PreProcessProjects - Just finished processing the projects")
}

// preProcessProject (3) is run before processing an individual project
func (b BuildPluginImpl) PreProcessProject(url *string, path *string, artifacts *string, branches *[]string, scripts *[]string) {
	fmt.Println("Example Plugin: PreProcessProjects - Just about to start processing an individual project known as", *path)
}

// postProcessProject (8) is run after processing an individual project
func (b BuildPluginImpl) PostProcessProject(url *string, path *string, artifacts *string, branches *[]string, scripts *[]string) {
	fmt.Println("Example Plugin: PostProcessProject - Just finished processing an individual project known as", *path)
}

// preProcessBranch (4) is run before processing a branch within a project
func (b BuildPluginImpl) PreProcessBranch(projectDir *string, branchName *string, workDirDesc *string) {
	fmt.Println("Example Plugin: PreProcessBranch - Just about to start processing a branch of an individual project, the branch is", *branchName, "and the working directory is described as", *workDirDesc)
}

// postProcessBranch (7) is run after processing a branch within a project
func (b BuildPluginImpl) PostProcessBranch(projectDir *string, branchName *string, workDirDesc *string) {
	fmt.Println("Example Plugin: PostProcessBranch - Just finished processing a branch of an individual project, the branch was", *branchName, "and the working directory is now described as", *workDirDesc)
}

// preProcessArtifacts (5) is run before processing the build artifacts of a branch
func (b BuildPluginImpl) PreProcessArtifacts(artifactPath *string, projectPath *string, branchName *string) {
	fmt.Println("Example Plugin: PreProcessArtifacts - Just about to start processing artifacts for a branch of an individual project, the project is", *projectPath, ", branch", *branchName, "and the artifacts will be in", *artifactPath)
}

// postProcessArtifacts (6) is run after processing the build artifacts of a branch
func (b BuildPluginImpl) PostProcessArtifacts(artifactPath *string, projectPath *string, branchName *string) {
	fmt.Println("Example Plugin: PreProcessArtifacts - Just finished processing artifacts for a branch of an individual project, the project was", *projectPath, ", branch", *branchName, "and the artifacts will be in", *artifactPath)
}

var BuildPlugin BuildPluginImpl
