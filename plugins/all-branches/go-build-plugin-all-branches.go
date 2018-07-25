// go-build-plugin-all-branches
package main

import (
	"fmt"
	"strings"
	"os/exec"
)

type BuildPluginImpl struct{}

var abWorkDir string

// pluginInit (0) is the Plugin Initialiser, called on load of plugin file
func (b BuildPluginImpl) PluginInit(rawConfig []byte) error {
	fmt.Println("All-Branches Plugin Initialised.")
	return nil
}

// postLoadPlugins (1) is the first fully-loaded hook, after all plugins are loaded
func (b BuildPluginImpl) PostLoadPlugins(version *string, buildTime *string) {
	fmt.Println("All-Branches Plugin running against core version", *version)
}

// preProcessProjects (2) is run before processing all projects
func (b BuildPluginImpl) PreProcessProjects(workingDir *string, homeDir *string, async *bool) {
	fmt.Println("All-Branches Plugin: Using configured home directory ", *homeDir)
	abWorkDir = *homeDir
}

// postProcessProjects (9) is run after processing all projects
func (b BuildPluginImpl) PostProcessProjects(workingDir *string, homeDir *string, async *bool) { }

// preProcessProject (3) is run before processing an individual project
func (b BuildPluginImpl) PreProcessProject(url *string, path *string, artifacts *string, branches *[]string, scripts *[]string) {

	// Slice pointers aren't indexed, so we need to copy the contents here for now
	confBranches := *branches

	if confBranches[0] != "*" {
		fmt.Println("All-Branches Plugin: Project", *path, "is NOT configured for all branches.")
	}

	fmt.Println("All-Branches Plugin: Enumerating remote branches for configured project", *path)

	parts := strings.Fields("git ls-remote --heads -q")
	cmd := exec.Command(parts[0], parts[1:]...)

	cmd.Dir = abWorkDir + "/projects/" + *path

	data, err := cmd.Output()
	if err != nil {
		fmt.Println("All-Branches Plugin: Failed to enumerate remote branches for project", *path)
		fmt.Println(err)
		return
	}

	rawResult := string(data[:])

	lines := 0
	var newBranches []string
	for _, line := range strings.Split(strings.TrimSuffix(rawResult, "\n"), "\n") {

		// Split the hash and ref
		refs := strings.Split(line, "\t")

		// Get the branch name only
		branchName := strings.TrimPrefix(refs[1], "refs/heads/")

		fmt.Println("All-Branches Plugin: Project", *path, "-- Found branch ", branchName, "(Remote ref:", refs[0], refs[1], ")")

		newBranches = append(newBranches, branchName)
		lines++
	}

	fmt.Println("All-Branches Plugin: Project", *path, "has", lines, "branches available on the remote.")
	*branches = newBranches
}

// postProcessProject (8) is run after processing an individual project
func (b BuildPluginImpl) PostProcessProject(url *string, path *string, artifacts *string, branches *[]string, scripts *[]string) { }

// preProcessBranch (4) is run before processing a branch within a project
func (b BuildPluginImpl) PreProcessBranch(projectDir *string, branchName *string, workDirDesc *string) { }

// postProcessBranch (7) is run after processing a branch within a project
func (b BuildPluginImpl) PostProcessBranch(projectDir *string, branchName *string, workDirDesc *string) { }

// preProcessArtifacts (5) is run before processing the build artifacts of a branch
func (b BuildPluginImpl) PreProcessArtifacts(artifactPath *string, projectPath *string, branchName *string) { }

// postProcessArtifacts (6) is run after processing the build artifacts of a branch
func (b BuildPluginImpl) PostProcessArtifacts(artifactPath *string, projectPath *string, branchName *string) { }

var BuildPlugin BuildPluginImpl
