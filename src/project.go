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

// project - Project processing
package main

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/libgit2/git2go"
	"github.com/getsentry/raven-go"
	"strconv"
	"path/filepath"
)

type scriptVariables struct {
	Project   string
	Branch    string
	URL       string
	Artifacts string
}

var pwd string

func processProjects(config *Configuration, cloneOpts *git.CloneOptions) {

	Log.Debug("Configuring WaitGroup")
	var w sync.WaitGroup
	w.Add(len(config.Projects))

	Log.Infof("Running from \"%s\" with configured home directory \"%s\".\n", pwd, config.Home)

	if config.Async == true {
		Log.Debug("Asynchronous Mode Enabled: Projects will be built in parallel.")
	}

	for _, proj := range config.Projects {
		if config.Async == true {
			// Async enabled, use goroutines :-
			go func(config *Configuration, proj ProjectConfig, cloneOpts *git.CloneOptions) {
				defer func() {
					if r := recover(); r != nil {
						if _, ok := r.(runtime.Error); ok {
							Log.Critical("Processing Project", proj.Path, "caused a runtime error:", r)
							panic(r)
						}
						Log.Error("Processing Project", proj.Path, "failed:", r)
					} else {
						Log.Info("Processing Project", proj.Path, "completed")
					}
				}()
				defer w.Done()
				Log.Infof("Processing project \"%s\" from url: \"%s\" in asynchronous mode.\n", proj.Path, proj.URL)
				runPreProcessProject(&proj.URL, &proj.Path, &proj.Artifacts, &proj.Branches, &proj.Scripts)
				processRepo(config, proj, cloneOpts)
				runPostProcessProject(&proj.URL, &proj.Path, &proj.Artifacts, &proj.Branches, &proj.Scripts)
			}(config, proj, cloneOpts)
		} else {
			// Async disabled, run normally in loop :-(
			Log.Debug("Asynchronous Mode Disabled: Projects will be built in sequence.")
			Log.Infof("Processing project \"%s\" from url: \"%s\".\n", proj.Path, proj.URL)
			runPreProcessProject(&proj.URL, &proj.Path, &proj.Artifacts, &proj.Branches, &proj.Scripts)
			processRepo(config, proj, cloneOpts)
			runPostProcessProject(&proj.URL, &proj.Path, &proj.Artifacts, &proj.Branches, &proj.Scripts)
		}
	}

	if config.Async == true {
		w.Wait()
	}

	Log.Info("Finished processing all configured projects.")
}

func processRepo(config *Configuration, proj ProjectConfig, cloneOpts *git.CloneOptions) {
	var repo *git.Repository
	var twd string
	fresh := false

	pStart := time.Now()

	Log.Debugf(" [%s] - checking for existing clone...\n", proj.Path)

	// Target working directory for this repo
	twd = config.Home + "/projects/" + proj.Path

	if _, err := os.Stat(twd); os.IsNotExist(err) {
		Log.Infof(" [%s] - project at \"%s\" does not exist, creating clone...\n", proj.Path, twd)
		repo, err = cloneRepo(twd, proj.URL, proj.Path, cloneOpts)
		if err != nil {
			raven.CaptureErrorAndWait(err, nil)
			Log.Critical(err)
			panic(err)
		}
		fresh = true
	}

	if _, err := os.Stat(twd); err == nil {
		Log.Infof(" [%s] - opening repository in \"%s\"...\n", proj.Path, twd)
		repo, err = git.OpenRepository(twd)
		if err != nil {
			raven.CaptureErrorAndWait(err, nil)
			Log.Critical(err)
			panic(err)
		}
	} else {
		Log.Debugf(" [%s] - error opening repository in \"%s\"\n", proj.Path, twd)
		raven.CaptureErrorAndWait(err, nil)
		Log.Critical(err)
		panic(err)
	}

	Log.Debugf(" [%s] - loading repository configuration...\n", proj.Path)

	repoConfig, err := repo.Config()
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		Log.Critical(err)
		panic(err)
	}
	defer repoConfig.Free()

	Log.Debugf(" [%s] - enabling remote origin pruning...\n", proj.Path)
	repoConfig.SetBool("remote.origin.prune", true)

	Log.Debugf(" [%s] - testing repository type (isBare)...\n", proj.Path)
	if repo.IsBare() {
		Log.Debugf(" [%s] - bare repository loaded and configured\n", proj.Path)
	} else {
		Log.Debugf(" [%s] - repository loaded and configured\n", proj.Path)
	}

	if fresh != true {
		// This isn't a fresh clone, but an existing repo. Fetch changes...
		Log.Debugf(" [%s] - fetching changes from remote...\n", proj.Path)
		err = fetchChanges(repo, proj.URL, proj.Path)
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf(" [%s] - failed to fetch changes from remote:\n", proj.Path)
			Log.Critical(err)
		}

		Log.Debugf(" [%s] - pulling changes from remote...\n", proj.Path)
		err = pullChanges(repo, proj.Path)
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf(" [%s] - failed to pull changes from remote:\n", proj.Path)
			Log.Critical(err)
		}
	}

	Log.Debugf(" [%s] - loading object database\n", proj.Path)

	odb, err := repo.Odb()
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		Log.Critical(err)
		panic(err)
	}

	Log.Debugf(" [%s] - counting objects\n", proj.Path)

	odblen := 0
	err = odb.ForEach(func(oid *git.Oid) error {
		odblen++
		return nil
	})
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		Log.Critical(err)
		panic(err)
	}

	Log.Debugf(" [%s] - object database loaded, %d objects.\n", proj.Path, odblen)

	Log.Debugf(" [%s] - loading branch processing configuration...\n", proj.Path)
	if proj.Branches[0] == "*" {
		Log.Debugf(" [%s] - project is configured to have all branches built.\n", proj.Path)
		proj.Branches = []string{"master", "develop"}
		Log.Warningf(" [%s] - project is set for wildcard branch build, but it is not yet supported; Only master and develop will be built.\n", proj.Path)
	} else {
		Log.Debugf(" [%s] - project is configured to have the following branches built: %s\n", proj.Path, strings.Join(proj.Branches[:], ", "))
	}


	processedBranches := 0

	for _, branchName := range proj.Branches {
		processedBranches++
		Log.Infof(" [%s] - processing branch %d \"%s\"...\n", proj.Path, processedBranches, branchName)
		bStart := time.Now()
		processBranch(config, proj, twd, branchName, repo)
		Log.Infof(" [%s] - completed branch %d \"%s\" in: %s\n", proj.Path, processedBranches, branchName, time.Since(bStart))
	}

	Log.Infof(" [%s] - completed %d branches in: %s\n", proj.Path, processedBranches, time.Since(pStart))
}

func processBranch(config *Configuration, proj ProjectConfig, twd string, branchName string, repo *git.Repository) {

	Log.Debugf(" [%s] - running project scripts...\n", proj.Path)

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				Log.Critical("Processing project", proj.Path, "branch", branchName, "caused a runtime error:", r)
				panic(r)
			}
			Log.Error("Processing project", proj.Path, "branch", branchName, "failed:", r)
		} else {
			Log.Info("Processing project", proj.Path, "branch", branchName, "completed.")
		}
	}()


	Log.Debugf(" [%s] - checking out branch \"%s\"...\n", proj.Path, branchName)
	coErr := checkoutBranch(repo, branchName)
	if coErr != nil {
		raven.CaptureErrorAndWait(coErr, nil)
		Log.Errorf(" [%s] - failed to checkout branch %s:\n", proj.Path, branchName)
		Log.Critical(coErr)
		panic(coErr)
	}

	Log.Infof(" [%s] - pulling changes from remote for branch %s...\n", proj.Path, branchName)
	pullErr := pullChanges(repo, proj.Path)
	if pullErr != nil {
		raven.CaptureError(pullErr, nil)
		Log.Errorf(" [%s] - failed to pull changes from remote for branch %s:\n", proj.Path, branchName)
		Log.Critical(pullErr)
	}

	description, descErr := describeWorkDir(repo, proj.Path)
	if descErr != nil {
		Log.Errorf(" [%s] - failed to describe working directory state post-checkout for branch %s:\n", proj.Path, branchName)
		Log.Error(descErr)
	}
	if description != "" {
		Log.Infof(" [%s] - on branch \"%s\", working directory is %s\n", proj.Path, branchName, description)
	}

	runPreProcessBranch(&twd, &branchName, &description)

	runProjectScripts(twd, branchName, proj)

	Log.Debugf(" [%s] - configuring artifacts pick-up path...\n", proj.Path)
	artifacts := twd + "/" + proj.Artifacts

	if _, afErr := os.Stat(artifacts); os.IsNotExist(afErr) {
		Log.Warningf(" [%s] ! build artifacts could not be found, maybe the build failed?\n", proj.Path)
		Log.Infof(" [%s] ! expected build artifacts in: \"%s\"\n", proj.Path, artifacts)
		Log.Noticef(" [%s] ! no build will be published for this project/branch.\n", proj.Path)
		return
	}

	if _, err := os.Stat(artifacts); err == nil {
		Log.Debugf(" [%s] - build artifacts found in: \"%s\"...\n", proj.Path, artifacts)
	}

	Log.Debugf(" [%s] - processing artifacts from pick-up location...\n", proj.Path)
	runPreProcessArtifacts(&artifacts, &proj.Path, &branchName)
	processArtifacts(config.Home, twd, artifacts, proj.Path, branchName)
	runPostProcessArtifacts(&artifacts, &proj.Path, &branchName)

	runPostProcessBranch(&twd, &branchName, &description)
}

func runProjectScripts(dir string, branchName string, proj ProjectConfig) {
	Log.Debugf(" [%s] - project has %d scripts configured\n", proj.Path, len(proj.Scripts))

	scriptIndex := 0
	for _, script := range proj.Scripts {

		// Setup the variables that can be substituted in the script for this run
		Log.Debugf(" [%s] - preparing project script %d: \"%s\"...\n", proj.Path, scriptIndex, script)

		scriptSubs := scriptVariables{proj.Path, branchName, proj.URL, proj.Artifacts}

		tmpl, err := template.New("script").Parse(script)
		scriptFinal := &bytes.Buffer{}
		tmpl.Execute(scriptFinal, scriptSubs)
		if err != nil {
			Log.Critical(err)
		}
		scriptFinalStr := scriptFinal.String()

		Log.Debugf(" [%s] - executing project script %d: \"%s\"...\n", proj.Path, scriptIndex, scriptFinalStr)

		stdout, stderr, err := execInDir(dir, scriptFinalStr)
		writeProjectLogs(stdout, stderr, scriptIndex, dir)
		if err != nil {
			Log.Debugf(" [%s] - error executing project script %d: \"%s\"...\n", proj.Path, scriptIndex, scriptFinalStr)
			Log.Debugf("%s\n", string(stdout))
			Log.Errorf("%s\n", string(stderr))
			Log.Critical(err)
			panic(err)
		}

		scriptIndex++
	}
}

func execInDir(dir string, command string) (string, string, error) {

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	parts := strings.Fields(command)

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = dir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		raven.CaptureError(err, nil)
		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}

func writeProjectLogs( stdout string, stderr string, index int, dir string ) {
	// Open the output log for writing
	soLogFile, soErr := os.Create( dir + "/go-build-stdout_" + strconv.Itoa(index) + ".log" )
	if soErr != nil {
		// Fatal error
		raven.CaptureErrorAndWait(soErr, nil)
		Log.Critical(soErr)
	}

	// Write the output to the file
	_, soErr = soLogFile.WriteString(stdout)
	if soErr != nil {
		// Fatal error
		raven.CaptureErrorAndWait(soErr, nil)
		Log.Critical(soErr)
	}

	soLogFile.Close()

	// Open the error log for writing
	seLogFile, seErr := os.Create( dir + "/go-build-stderr_" + strconv.Itoa(index) + ".log" )
	if seErr != nil {
		// Fatal error
		raven.CaptureErrorAndWait(seErr, nil)
		Log.Critical(seErr)
	}

	// Write the output to the file
	_, seErr = seLogFile.WriteString(stderr)
	if seErr != nil {
		// Fatal error
		raven.CaptureErrorAndWait(seErr, nil)
		Log.Critical(seErr)
	}

	seLogFile.Close()
}

func processArtifacts(home string, projectDir string, artifacts string, project string, branchName string) {
	Log.Infof(" [%s] - processing build artifacts for project \"%s\", branch \"%s\".\n", project, project, branchName)

	destination := home + "/artifacts/" + project + "/" + branchName
	destParts := strings.Split(destination, "/")
	destParent := strings.Join(destParts[:len(destParts)-1], "/")

	Log.Debugf(" [%s] - build artifacts will be stored in: \"%s\".\n", project, destination)

	Log.Debugf(" [%s] - removing any previous artifacts from the destination\n", project)
	rmErr := os.RemoveAll(destination)
	if rmErr != nil {
		raven.CaptureErrorAndWait(rmErr, nil)
		Log.Critical(rmErr)
		panic(rmErr)
	}

	Log.Debugf(" [%s] - creating destination directory structure\n", project)
	mkErr := os.MkdirAll(destParent, 0755)
	if mkErr != nil {
		raven.CaptureErrorAndWait(mkErr, nil)
		Log.Critical(mkErr)
		panic(mkErr)
	}

	Log.Debugf(" [%s] - moving build artifacts into destination\n", project)
	mvErr := os.Rename(artifacts, destination)
	if mvErr != nil {
		raven.CaptureErrorAndWait(mvErr, nil)
		Log.Critical(mvErr)
		panic(mvErr)
	}

	logGlob := projectDir + "/*.log"
	Log.Debugf(" [%s] - searching for build logs using glob: \"%s\"\n", project, logGlob)
	logFiles, lfErr := filepath.Glob(logGlob)
	if lfErr != nil {
		raven.CaptureErrorAndWait(lfErr, nil)
		Log.Critical(lfErr)
		panic(lfErr)
	}

	Log.Debugf(" [%s] - project has %d log files\n", project, len(logFiles))

	for _, f := range logFiles {
		lFile := filepath.Base(f)
		Log.Debugf(" [%s] - moving log file \"%s\" to destination\n", project, lFile)
		mvLfErr := os.Rename(f, destination + "/" + lFile)
			if mvLfErr != nil {
			raven.CaptureErrorAndWait(mvLfErr, nil)
			Log.Critical(mvLfErr)
			panic(mvLfErr)
		}
	}

	Log.Debugf(" [%s] - artifact processing completed.\n", project)
}
