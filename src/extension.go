package main

// BuildPlugin defines a common interface for go-build plugins
type BuildPlugin interface {
	// pluginInit 0. Plugin Initialiser, called on load of plugin file
	// receives the raw configuration as a byte array (to be parsed with json.Unmarshal)
	PluginInit([]byte) error

	// PostLoadPlugins 1. First hook, after plugins are loaded
	// plugins receive the core's version and build time to check compatibility here
	PostLoadPlugins(string, string)

	// PreProcessProjects 2. Before processing all projects
	// plugins receive the workingDir, homeDir, and async status
	PreProcessProjects(string, string, bool)

	// PostProcessProjects 9. After processing all projects
	// plugins receive the workingDir, homeDir, and async status
	PostProcessProjects(string, string, bool)

	// PreProcessProject 3. Before processing an individual project
	// url, path, artifacts, branches, scripts
	PreProcessProject(string, string, string, []string, []string)

	// PostProcessProject 8. After processing an individual project
	// url, path, artifacts, branches, scripts
	PostProcessProject(string, string, string, []string, []string)

	// PreProcessBranch 4. Before processing a branch within a project
	// projectDir, branchName, workDirDesc
	PreProcessBranch(string, string, string)

	// PostProcessBranch 7. After processing a branch within a project
	// projectDir, branchName, workDirDesc
	PostProcessBranch(string, string, string)

	// PreProcessArtifacts 5. Before processing the build artifacts of a branch
	// artifactPath, projectPath, branchName
	PreProcessArtifacts(string, string, string)

	// PostProcessArtifacts 6. After processing the build artifacts of a branch
	// artifactPath, projectPath, branchName
	PostProcessArtifacts(string, string, string)
}
