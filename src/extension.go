package main

// BuildPlugin defines a common interface for go-build plugins
type BuildPlugin interface {
	// pluginInit 0. Plugin Initialiser, called on load of plugin file
	PluginInit() error

	// PostLoadPlugins 1. First hook, after plugins are loaded
	PostLoadPlugins()

	// PreProcessProjects 2. Before processing all projects
	PreProcessProjects()

	// PostProcessProjects 9. After processing all projects
	PostProcessProjects()

	// PreProcessProject 3. Before processing an individual project
	PreProcessProject()

	// PostProcessProject 8. After processing an individual project
	PostProcessProject()

	// PreProcessBranch 4. Before processing a branch within a project
	PreProcessBranch()

	// PostProcessBranch 7. After processing a branch within a project
	PostProcessBranch()

	// PreProcessArtifacts 5. Before processing the build artifacts of a branch
	PreProcessArtifacts()

	// PostProcessArtifacts 6. After processing the build artifacts of a branch
	PostProcessArtifacts()
}
