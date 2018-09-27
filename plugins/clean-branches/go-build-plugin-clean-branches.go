// go-build-plugin-clean-branches
package main

import (
	"fmt"
	"bytes"
	"strings"
	"os/exec"
)

type BuildPluginImpl struct{}

// pluginInit (0) is the Plugin Initialiser, called on load of plugin file
func (b BuildPluginImpl) PluginInit(rawConfig []byte) error {
	fmt.Println("Clean Branches Plugin: Clean Branches Plugin Initialised.")
	return nil
}

// postLoadPlugins (1) is the first fully-loaded hook, after all plugins are loaded
func (b BuildPluginImpl) PostLoadPlugins(version *string, buildTime *string) {}

// preProcessProjects (2) is run before processing all projects
func (b BuildPluginImpl) PreProcessProjects(workingDir *string, homeDir *string, async *bool) {}

// postProcessProjects (9) is run after processing all projects
func (b BuildPluginImpl) PostProcessProjects(workingDir *string, homeDir *string, async *bool) {}

// preProcessProject (3) is run before processing an individual project
func (b BuildPluginImpl) PreProcessProject(url *string, path *string, artifacts *string, branches *[]string, scripts *[]string) {}

// postProcessProject (8) is run after processing an individual project
func (b BuildPluginImpl) PostProcessProject(url *string, path *string, artifacts *string, branches *[]string, scripts *[]string) {}

// preProcessBranch (4) is run before processing a branch within a project
func (b BuildPluginImpl) PreProcessBranch(projectDir *string, branchName *string, workDirDesc *string) {
	fmt.Println("Clean Branches Plugin: PreProcessBranch - Cleaning project for", *branchName)

	command := "git clean -d -f"

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	parts := strings.Fields(command)

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = *projectDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	fmt.Println(stdout.String())
	fmt.Println(stderr.String())

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Clean Branches Plugin: PreProcessBranch - Cleaning completed for branch", *branchName)
}

// postProcessBranch (7) is run after processing a branch within a project
func (b BuildPluginImpl) PostProcessBranch(projectDir *string, branchName *string, workDirDesc *string) {}

// preProcessArtifacts (5) is run before processing the build artifacts of a branch
func (b BuildPluginImpl) PreProcessArtifacts(artifactPath *string, projectPath *string, branchName *string) {}

// postProcessArtifacts (6) is run after processing the build artifacts of a branch
func (b BuildPluginImpl) PostProcessArtifacts(artifactPath *string, projectPath *string, branchName *string) {}

var BuildPlugin BuildPluginImpl
