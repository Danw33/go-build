/**
Dan's Multi-Project Builder
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

package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/libgit2/git2go"
	"github.com/op/go-logging"
)

type configuration struct {
	Home     string    `json:"home"`
	Async    bool      `json:"async"`
	Log      logger    `json:"log"`
	Projects []project `json:"projects"`
}

type logger struct {
	Level string `json:"level"`
}

type project struct {
	URL       string   `json:"url"`
	Path      string   `json:"path"`
	Artifacts string   `json:"artifacts"`
	Branches  []string `json:"branches"`
	Scripts   []string `json:"scripts"`
}

var log = logging.MustGetLogger("example")

var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

func main() {
	start := time.Now()

	// Setup logger, defualt to INFO level
	logBackend := logging.NewLogBackend(os.Stdout, "", 0)
	logBackendFormatted := logging.NewBackendFormatter(logBackend, format)
	logging.SetBackend(logBackendFormatted)
	logging.SetLevel(logging.INFO, "")

	// Check for verbose flag, if it's present, up the level to DEBUG
	verbose := false
	for _, arg := range os.Args {
		if arg == "-v" {
			verbose = true
			logging.SetLevel(logging.DEBUG, "")
		}
	}

	log.Info("go-build: Danw33's Multi-Project Build Utility")
	log.Infof("Running on OS: \"%s\", Architecture: \"%s\"\n", runtime.GOOS, runtime.GOARCH)

	log.Debug("Reading configuration file...")
	configFile := ".build.json"
	cfgByte, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Critical(err)
		panic(err)
	}
	cfg := string(cfgByte)
	config := parseConfig(cfg)

	// Adjust the log level agian, this time from the confguration file, but only if verbose isn't passed
	if verbose == false {
		level, err := logging.LogLevel(config.Log.Level)
		if err != nil {
			log.Critical(err)
		}
		logging.SetLevel(level, "")
	}

	log.Infof("Configuration Loaded.")

	cloneOpts := configureCloneOpts()

	log.Debug("Starting Project Processor...")
	processProjects(config, cloneOpts)

	log.Infof("All projects completed in: %s", time.Since(start))
}

func processProjects(config *configuration, cloneOpts *git.CloneOptions) {

	log.Debug("Configuring WaitGroup")
	var w sync.WaitGroup
	w.Add(len(config.Projects))

	log.Debug("Detecting Working Directory")
	pwd, err := os.Getwd()
	if err != nil {
		log.Critical(err)
		panic(err)
	}

	log.Infof("Running from \"%s\" with configured home directory \"%s\".\n", pwd, config.Home)

	if config.Async == true {
		log.Debug("Asynchronous Mode Enabled: Projects will be built in parallel.")
	}

	for _, proj := range config.Projects {
		if config.Async == true {
			// Async enabled, use goroutines :-
			go func(config *configuration, proj project, cloneOpts *git.CloneOptions) {
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

func processRepo(config *configuration, proj project, cloneOpts *git.CloneOptions) {
	var repo *git.Repository
	var twd string

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

		log.Infof(" [%s] - on branch \"%s\", processing...\n", proj.Path, branchName)
		processBranch(config, proj, twd, branchName)
		log.Infof(" [%s] - completed branch \"%s\" in: %s\n", proj.Path, branchName, time.Since(bStart))
	}

	log.Infof(" [%s] - completed all configured branches in: %s\n", proj.Path, time.Since(pStart))
}

func processBranch(config *configuration, proj project, twd string, branchName string) {

	log.Debugf(" [%s] - running project scripts...\n", proj.Path)

	runProjectScripts(twd, proj)

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

func runProjectScripts(dir string, proj project) {
	log.Debugf(" [%s] - project has %d scripts configured\n", proj.Path, len(proj.Scripts))

	for _, script := range proj.Scripts {
		log.Debugf(" [%s] - executing project script: \"%s\"...\n", proj.Path, script)
		stdout, err := execInDir(dir, script)
		if err != nil {
			log.Debugf(" [%s] - error executing project script: \"%s\"...\n", proj.Path, script)
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

func parseConfig(cfg string) *configuration {
	res := configuration{}
	log.Debug("Parsing Configuration using json.Unmarshal...\n")
	json.Unmarshal([]byte(cfg), &res)
	log.Debugf("Loaded Configuration: %d Projects Configured.\n", len(res.Projects))
	return &res
}

func configureCloneOpts() *git.CloneOptions {
	remoteCallbacks := git.RemoteCallbacks{
		CredentialsCallback:      credentialsCallback,
		CertificateCheckCallback: certificateCheckCallback,
	}

	cloneOpts := &git.CloneOptions{
		FetchOptions: &git.FetchOptions{
			RemoteCallbacks: remoteCallbacks,
		},
		Bare: false,
	}

	return cloneOpts
}

func credentialsCallback(url string, username string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
	log.Debugf(" [git] - running credentials callback with username \"%s\" for url \"%s\"\n", username, url)
	ret, cred := git.NewCredSshKeyFromAgent(username)
	return git.ErrorCode(ret), &cred
}

func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	log.Debugf(" [git] - running certificate check callback for hostname \"%s\"\n", hostname)
	if hostname != "github.com" && hostname != "gitlab.com" {
		log.Debugf(" [git] - certificate check callback passed for hostname \"%s\"\n", hostname)
		return git.ErrUser
	}
	log.Warningf(" [git] - certificate check callback passed for hostname \"%s\"\n", hostname)
	return 0
}

func cloneRepo(twd string, url string, path string, cloneOpts *git.CloneOptions) (*git.Repository, error) {

	log.Debugf(" [%s] - cloning repository from \"%s\" into \"%s\"\n", path, url, twd)

	// Clone
	repo, err := git.Clone(url, twd, cloneOpts)
	if err != nil {
		return nil, err
	}

	log.Debugf(" [%s] - clone completed, finding head ref\n", path)

	// Get HEAD ref
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}

	log.Debugf(" [%s] - head is now at %v\n", path, head.Target())

	return repo, nil
}

func checkoutBranch(repo *git.Repository, branchName string) error {
	checkoutOpts := &git.CheckoutOpts{
		Strategy: git.CheckoutSafe | git.CheckoutRecreateMissing | git.CheckoutAllowConflicts | git.CheckoutUseTheirs,
	}

	//Getting the reference for the remote branch
	remoteBranch, err := repo.LookupBranch("origin/"+branchName, git.BranchRemote)
	if err != nil {
		log.Error("Failed to find remote branch: " + branchName)
		return err
	}
	defer remoteBranch.Free()

	// Lookup for commit from remote branch
	commit, err := repo.LookupCommit(remoteBranch.Target())
	if err != nil {
		log.Error("Failed to find remote branch commit: " + branchName)
		return err
	}
	defer commit.Free()

	localBranch, err := repo.LookupBranch(branchName, git.BranchLocal)
	// No local branch, lets create one
	if localBranch == nil || err != nil {
		// Creating local branch
		localBranch, err = repo.CreateBranch(branchName, commit, false)
		if err != nil {
			log.Error("Failed to create local branch: " + branchName)
			return err
		}

		// Setting upstream to origin branch
		err = localBranch.SetUpstream("origin/" + branchName)
		if err != nil {
			log.Error("Failed to create upstream to origin/" + branchName)
			return err
		}
	}
	if localBranch == nil {
		return errors.New("Error while locating/creating local branch")
	}
	defer localBranch.Free()

	// Getting the tree for the branch
	localCommit, err := repo.LookupCommit(localBranch.Target())
	if err != nil {
		log.Error("Failed to lookup for commit in local branch " + branchName)
		return err
	}
	defer localCommit.Free()

	tree, err := repo.LookupTree(localCommit.TreeId())
	if err != nil {
		log.Error("Failed to lookup for tree " + branchName)
		return err
	}
	defer tree.Free()

	// Checkout the tree
	err = repo.CheckoutTree(tree, checkoutOpts)
	if err != nil {
		log.Error("Failed to checkout tree " + branchName)
		return err
	}

	// Setting the Head to point to our branch
	repo.SetHead("refs/heads/" + branchName)
	return nil
}
