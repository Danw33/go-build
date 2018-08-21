// go-build-plugin-example
package main

import (
	"fmt"
	"net/http"
	"os"
	"io"
	"strconv"
	"strings"
	"time"
	"os/exec"
)

type BuildPluginImpl struct{}

var coreVersion string
var homeDirectory string
var artifactsDirectory string
var counterProjects int
var counterBranches int

// pluginInit (0) is the Plugin Initialiser, called on load of plugin file
func (b BuildPluginImpl) PluginInit(rawConfig []byte) error {
	counterProjects = 0
	counterBranches = 0
	fmt.Println("Indexes Plugin: Index Generator Plugin Initialised.")
	return nil
}

// postLoadPlugins (1) is the first fully-loaded hook, after all plugins are loaded
func (b BuildPluginImpl) PostLoadPlugins(version *string, buildTime *string) {
	fmt.Println("Indexes Plugin: Index Generator running against core version", *version)
	coreVersion = *version
}

// preProcessProjects (2) is run before processing all projects
func (b BuildPluginImpl) PreProcessProjects(workingDir *string, homeDir *string, async *bool) {
	homeDirectory = *homeDir

	// This is currently a hard-coded subdirectory - See the core's processArtifacts method in project.go
	artifactsDirectory = homeDirectory + "/artifacts/"
}

// postProcessProjects (9) is run after processing all projects
func (b BuildPluginImpl) PostProcessProjects(workingDir *string, homeDir *string, async *bool) {
	apiBase := "https://img.shields.io/badge/"
	fileBase := artifactsDirectory + "go-build-badges"

	fmt.Println("Indexes Plugin: Creating badges via shields.io API")

	fmt.Println("Indexes Plugin: Removing any existing badges from", fileBase)
	rmErr := os.RemoveAll(fileBase)
	if rmErr != nil {
		fmt.Println(rmErr)
		panic(rmErr)
	}

	fmt.Println("Indexes Plugin: Creating destination directory structure for badges in", fileBase)
	mkErr := os.MkdirAll(fileBase, 0755)
	if mkErr != nil {
		fmt.Println(mkErr)
		panic(mkErr)
	}

	fmt.Println("Indexes Plugin: Making projects badge with value", counterProjects)
	indexPluginFetchBadge(apiBase + "projects-" + strconv.Itoa(counterProjects) + "-brightgreen.svg", fileBase + "/projects.svg")

	fmt.Println("Indexes Plugin: Making branches badge with value", counterBranches)
	indexPluginFetchBadge(apiBase + "branches-" + strconv.Itoa(counterBranches) + "-brightgreen.svg", fileBase + "/branches.svg")

	badgeSafeVersion := strings.Replace(coreVersion, "-", "--", -1)
	versionBadgeUrl := "go--build-" + badgeSafeVersion + "-ff69b4.svg?logo=github&logoColor=f1f1f1&link=https://github.com/Danw33/go-build&link=https://github.com/Danw33/go-build/releases"
	indexPluginFetchBadge(apiBase + versionBadgeUrl, fileBase + "/version.svg")

	badgeTime := time.Now()
	badgeDate := badgeTime.Format(time.RFC3339)
	badgeSafeDate :=   strings.Replace(badgeDate, "-", "--", -1)
	timestampBadgeUrl := "artifacts-" + badgeSafeDate + "-blue.svg"
	indexPluginFetchBadge(apiBase + timestampBadgeUrl, fileBase + "/artifacts.svg")

}

// preProcessProject (3) is run before processing an individual project
func (b BuildPluginImpl) PreProcessProject(url *string, path *string, artifacts *string, branches *[]string, scripts *[]string) {}

// postProcessProject (8) is run after processing an individual project
func (b BuildPluginImpl) PostProcessProject(url *string, path *string, artifacts *string, branches *[]string, scripts *[]string) {
	counterProjects++
}

// preProcessBranch (4) is run before processing a branch within a project
func (b BuildPluginImpl) PreProcessBranch(projectDir *string, branchName *string, workDirDesc *string) {}

// postProcessBranch (7) is run after processing a branch within a project
func (b BuildPluginImpl) PostProcessBranch(projectDir *string, branchName *string, workDirDesc *string) {
	counterBranches++
}

// preProcessArtifacts (5) is run before processing the build artifacts of a branch
func (b BuildPluginImpl) PreProcessArtifacts(artifactPath *string, projectPath *string, branchName *string) {}

// postProcessArtifacts (6) is run after processing the build artifacts of a branch
func (b BuildPluginImpl) PostProcessArtifacts(artifactPath *string, projectPath *string, branchName *string) {
	fmt.Println("Indexes Plugin: PreProcessArtifacts - Just finished processing artifacts for a branch of an individual project, the project was", *projectPath, ", branch", *branchName, "and the artifacts will be in", *artifactPath)
}

var BuildPlugin BuildPluginImpl

func indexPluginFetchBadge( url string, fileName string ) {

	parts := strings.Fields( "wget -O " + fileName + " " + url )
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = homeDirectory
	_, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
	}

	return

	// This is the right way to do it, but in testing it causes an irrecoverable panic:

	// Fetch the badge from the API
	response, e := http.Get(url)
	if e != nil {
		// Fatal error
		fmt.Println(e)
	}

	defer response.Body.Close()

	// Open the file for writing
	file, err := os.Create( fileName )
	if err != nil {
		// Fatal error
		fmt.Println(err)
	}

	// Use io.Copy to just dump the response body to the file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		// Fatal error
		fmt.Println(err)
	}

	file.Close()
}