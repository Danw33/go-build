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
)

type scriptVariables struct {
	Project   string
	Branch    string
	URL       string
	Artifacts string
}

var pwd string

func processProjects(config *Configuration, cloneOpts *git.CloneOptions) {

	log.Debug("Configuring WaitGroup")
	var w sync.WaitGroup
	w.Add(len(config.Projects))

	log.Infof("Running from \"%s\" with configured home directory \"%s\".\n", pwd, config.Home)

	if config.Async == true {
		log.Debug("Asynchronous Mode Enabled: Projects will be built in parallel.")
	}

	for _, proj := range config.Projects {
		if config.Async == true {
			// Async enabled, use goroutines :-
			go func(config *Configuration, proj ProjectConfig, cloneOpts *git.CloneOptions) {
				defer func() {
					if r := recover(); r != nil {
						if _, ok := r.(runtime.Error); ok {
							log.Critical("Processing Project", proj.Path, "caused a runtime error:", r)
							panic(r)
						}
						log.Error("Processing Project", proj.Path, "failed:", r)
					} else {
						log.Info("Processing Project", proj.Path, "completed")
					}
				}()
				defer w.Done()
				log.Infof("Processing project \"%s\" from url: \"%s\" in asynchronous mode.\n", proj.Path, proj.URL)
				processRepo(config, proj, cloneOpts)
			}(config, proj, cloneOpts)
		} else {
			// Async disabled, run normally in loop :-(
			log.Debug("Asynchronous Mode Disabled: Projects will be built in sequence.")
			log.Infof("Processing project \"%s\" from url: \"%s\".\n", proj.Path, proj.URL)
			processRepo(config, proj, cloneOpts)
		}
	}

	if config.Async == true {
		w.Wait()
	}

	log.Info("Finished processing all configured projects.")
}

func processRepo(config *Configuration, proj ProjectConfig, cloneOpts *git.CloneOptions) {
	var repo *git.Repository
	var twd string
	fresh := false

	pStart := time.Now()

	log.Debugf(" [%s] - checking for existing clone...\n", proj.Path)

	// Target working directory for this repo
	twd = config.Home + "/projects/" + proj.Path

	if _, err := os.Stat(twd); os.IsNotExist(err) {
		log.Infof(" [%s] - project at \"%s\" does not exist, creating clone...\n", proj.Path, twd)
		repo, err = cloneRepo(twd, proj.URL, proj.Path, cloneOpts)
		if err != nil {
			log.Critical(err)
			panic(err)
		}
		fresh = true
	}

	if _, err := os.Stat(twd); err == nil {
		log.Infof(" [%s] - opening repository in \"%s\"...\n", proj.Path, twd)
		repo, err = git.OpenRepository(twd)
		if err != nil {
			log.Critical(err)
			panic(err)
		}
	} else {
		log.Debugf(" [%s] - error opening repository in \"%s\"\n", proj.Path, twd)
		log.Critical(err)
		panic(err)
	}

	log.Debugf(" [%s] - loading repository configuration...\n", proj.Path)

	repoConfig, err := repo.Config()
	if err != nil {
		log.Critical(err)
		panic(err)
	}
	defer repoConfig.Free()

	log.Debugf(" [%s] - enabling remote origin pruning...\n", proj.Path)
	repoConfig.SetBool("remote.origin.prune", true)

	log.Debugf(" [%s] - testing repository type (isBare)...\n", proj.Path)
	if repo.IsBare() {
		log.Debugf(" [%s] - bare repository loaded and configured\n", proj.Path)
	} else {
		log.Debugf(" [%s] - repository loaded and configured\n", proj.Path)
	}

	if fresh != true {
		// This isn't a fresh clone, but an existing repo. Fetch changes...
		log.Debugf(" [%s] - fetching changes from remote...\n", proj.Path)
		err = fetchChanges(repo, proj.URL, proj.Path)
		if err != nil {
			log.Errorf(" [%s] - failed to fetch changes from remote:\n", proj.Path)
			log.Critical(err)
		}

		log.Debugf(" [%s] - pulling changes from remote...\n", proj.Path)
		err = pullChanges(repo, proj.Path)
		if err != nil {
			log.Errorf(" [%s] - failed to pull changes from remote:\n", proj.Path)
			log.Critical(err)
		}
	}

	log.Debugf(" [%s] - loading object database\n", proj.Path)

	odb, err := repo.Odb()
	if err != nil {
		log.Critical(err)
		panic(err)
	}

	log.Debugf(" [%s] - counting objects\n", proj.Path)

	odblen := 0
	err = odb.ForEach(func(oid *git.Oid) error {
		odblen++
		return nil
	})
	if err != nil {
		log.Critical(err)
		panic(err)
	}

	log.Debugf(" [%s] - object database loaded, %d objects.\n", proj.Path, odblen)

	log.Debugf(" [%s] - loading branch processing configuration...\n", proj.Path)
	if proj.Branches[0] == "*" {
		log.Debugf(" [%s] - project is configured to have all branches built.\n", proj.Path)
		proj.Branches = []string{"master", "develop"}
		log.Warningf(" [%s] - project is set for wildcard branch build, but it is not yet supported; Only master and develop will be built.\n", proj.Path)
	} else {
		log.Debugf(" [%s] - project is configured to have the following branches built: %s\n", proj.Path, strings.Join(proj.Branches[:], ", "))
	}

	for _, branchName := range proj.Branches {
		log.Debugf(" [%s] - checking out branch \"%s\"...\n", proj.Path, branchName)
		bStart := time.Now()
		err = checkoutBranch(repo, branchName)
		if err != nil {
			log.Critical(err)
			panic(err)
		}

		log.Infof(" [%s] - pulling changes from remote for branch %s...\n", proj.Path, branchName)
		err = pullChanges(repo, proj.Path)
		if err != nil {
			log.Errorf(" [%s] - failed to pull changes from remote for branch %s:\n", proj.Path, branchName)
			log.Critical(err)
		}

		description, err := describeWorkDir(repo, proj.Path)
		if err != nil {
			log.Errorf(" [%s] - failed to describe working directory state post-checkout for branch %s:\n", proj.Path, branchName)
			log.Error(err)
		}
		if description != "" {
			log.Infof(" [%s] - on branch \"%s\", working directory is %s\n", proj.Path, branchName, description)
		}

		log.Infof(" [%s] - processing branch \"%s\"...\n", proj.Path, branchName)
		processBranch(config, proj, twd, branchName)
		log.Infof(" [%s] - completed branch \"%s\" in: %s\n", proj.Path, branchName, time.Since(bStart))
	}

	log.Infof(" [%s] - completed all configured branches in: %s\n", proj.Path, time.Since(pStart))
}

func processBranch(config *Configuration, proj ProjectConfig, twd string, branchName string) {

	log.Debugf(" [%s] - running project scripts...\n", proj.Path)

	runProjectScripts(twd, branchName, proj)

	log.Debugf(" [%s] - configuring artifacts pick-up path...\n", proj.Path)
	artifacts := twd + "/" + proj.Artifacts

	if _, err := os.Stat(artifacts); os.IsNotExist(err) {
		log.Warningf(" [%s] ! build artifacts could not be found, maybe the build failed?\n", proj.Path)
		log.Infof(" [%s] ! expected build artifacts in: \"%s\"\n", proj.Path, artifacts)
		log.Noticef(" [%s] ! no build will be published for this project/branch.\n", proj.Path)
		return
	}

	if _, err := os.Stat(artifacts); err == nil {
		log.Debugf(" [%s] - build artifacts found in: \"%s\"...\n", proj.Path, artifacts)
	}

	log.Debugf(" [%s] - processing artifacts from pick-up location...\n", proj.Path)
	processArtifacts(config.Home, artifacts, proj.Path, branchName)

}

func runProjectScripts(dir string, branchName string, proj ProjectConfig) {
	log.Debugf(" [%s] - project has %d scripts configured\n", proj.Path, len(proj.Scripts))

	for _, script := range proj.Scripts {

		// Setup the variables that can be substituted in the script for this run
		log.Debugf(" [%s] - preparing project script: \"%s\"...\n", proj.Path, script)

		scriptSubs := scriptVariables{proj.Path, branchName, proj.URL, proj.Artifacts}

		tmpl, err := template.New("script").Parse(script)
		scriptFinal := &bytes.Buffer{}
		tmpl.Execute(scriptFinal, scriptSubs)
		if err != nil {
			log.Critical(err)
		}
		scriptFinalStr := scriptFinal.String()

		log.Debugf(" [%s] - executing project script: \"%s\"...\n", proj.Path, scriptFinalStr)

		stdout, err := execInDir(dir, scriptFinalStr)
		if err != nil {
			log.Debugf(" [%s] - error executing project script: \"%s\"...\n", proj.Path, scriptFinalStr)
			log.Debugf("%s\n", string(stdout))
			log.Critical(err)
			panic(err)
		}
	}
}

func execInDir(dir string, command string) ([]byte, error) {
	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = dir
	data, err := cmd.Output()
	if err != nil {
		return data, err
	}

	return data, nil
}

func processArtifacts(home string, artifacts string, project string, branchName string) {
	log.Infof(" [%s] - processing build artifacts for project \"%s\", branch \"%s\".\n", project, project, branchName)

	destParent := home + "/artifacts/" + project
	destination := destParent + "/" + branchName

	log.Debugf(" [%s] - build artifacts will be stored in: \"%s\".\n", project, destination)

	log.Debugf(" [%s] - removing any previous artifacts from the destination\n", project)
	err := os.RemoveAll(destination)
	if err != nil {
		log.Critical(err)
		panic(err)
	}

	log.Debugf(" [%s] - creating destination directory structure\n", project)
	err = os.MkdirAll(destParent, 0755)
	if err != nil {
		log.Critical(err)
		panic(err)
	}

	log.Debugf(" [%s] - moving build artifacts into destination\n", project)
	err = os.Rename(artifacts, destination)
	if err != nil {
		log.Critical(err)
		panic(err)
	}

	log.Debugf(" [%s] - artifact processing completed.\n", project)
}
