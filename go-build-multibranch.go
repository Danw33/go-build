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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/libgit2/git2go"
)

type configuration struct {
	Home     string    `json:"home"`
	Async    bool      `json:"async"`
	Projects []project `json:"projects"`
}

type project struct {
	URL       string   `json:"url"`
	Path      string   `json:"path"`
	Artifacts string   `json:"artifacts"`
	Branches  []string `json:"branches"`
	Scripts   []string `json:"scripts"`
}

func main() {
	log.Println("Dan's Multi-Project Builder")

	fmt.Println("Reading configuration file...")
	configFile := ".build.json"
	cfgByte, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	cfg := string(cfgByte)
	config := parseConfig(cfg)

	cloneOpts := configureCloneOpts()

	processProjects(config, cloneOpts)
}

func processProjects(config *configuration, cloneOpts *git.CloneOptions) {
	var w sync.WaitGroup
	w.Add(len(config.Projects))

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Running from \"%s\" with configured home directory \"%s\".\n", pwd, config.Home)

	for _, proj := range config.Projects {
		if config.Async == true {
			// Async enabled, use goroutines :-)
			go func(config *configuration, proj project, cloneOpts *git.CloneOptions) {
				defer w.Done()
				fmt.Printf("Processing project \"%s\" from url: \"%s\".\n", proj.Path, proj.URL)
				processRepo(config, proj, cloneOpts)
			}(config, proj, cloneOpts)
		} else {
			// Async disabled, run normally in loop :-(
			processRepo(config, proj, cloneOpts)
		}
	}

	if config.Async == true {
		w.Wait()
	}

	fmt.Println("Finished processing all configured projects.")
}

func processRepo(config *configuration, proj project, cloneOpts *git.CloneOptions) {
	var repo *git.Repository
	var twd string

	fmt.Printf(" [%s] - checking for existing clone...\n", proj.Path)

	// Target working directory for this repo
	twd = config.Home + "/" + proj.Path

	if _, err := os.Stat(twd); os.IsNotExist(err) {
		fmt.Printf(" [%s] - project at \"%s\" does not exist, creating clone...\n", proj.Path, twd)
		repo, err = cloneRepo(twd, proj.URL, proj.Path, cloneOpts)
		if err != nil {
			panic(err)
		}
	}

	if _, err := os.Stat(proj.Path); err == nil {
		fmt.Printf(" [%s] - opening repository in \"%s\"...\n", proj.Path, twd)
		repo, err = git.OpenRepository(twd)
		if err != nil {
			panic(err)
		}
	}

	repoConfig, err := repo.Config()
	if err != nil {
		panic(err)
	}
	defer repoConfig.Free()

	repoConfig.SetBool("remote.origin.prune", true)

	if repo.IsBare() {
		fmt.Printf(" [%s] - bare repository loaded and configured\n", proj.Path)
	} else {
		fmt.Printf(" [%s] - repository loaded and configured\n", proj.Path)
	}

	fmt.Printf(" [%s] - loading object database\n", proj.Path)

	odb, err := repo.Odb()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(" [%s] - counting objects\n", proj.Path)

	odblen := 0
	err = odb.ForEach(func(oid *git.Oid) error {
		odblen++
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(" [%s] - object database loaded, %d objects.\n", proj.Path, odblen)

	//	printRepoDebug(repo)

	if proj.Branches[0] == "*" {
		fmt.Printf(" [%s] - project is configured to have all branches built.\n", proj.Path)
		proj.Branches = []string{"master", "develop"}
	} else {
		fmt.Printf(" [%s] - project is configured to have the following branches built: %s\n", proj.Path, strings.Join(proj.Branches[:], ", "))
	}

	for _, branchName := range proj.Branches {
		fmt.Printf(" [%s] - checking out branch \"%s\"...\n", proj.Path, branchName)
		err = checkoutBranch(repo, branchName)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf(" [%s] - on branch \"%s\", processing...\n", proj.Path, branchName)
		processBranch(config, proj, twd, branchName)
	}

}

func processBranch(config *configuration, proj project, twd string, branchName string) {

	fmt.Printf(" [%s] - running project scripts...\n", proj.Path)

	runProjectScripts(twd, proj)

	artifacts := twd + "/" + proj.Artifacts

	if _, err := os.Stat(artifacts); os.IsNotExist(err) {
		fmt.Printf(" [%s] ! build artifacts could not be found, maybe the build failed?\n", proj.Path)
		fmt.Printf(" [%s] ! expected build artifacts in: \"%s\"\n", proj.Path, artifacts)
		fmt.Printf(" [%s] ! no build will be published for this project/branch.\n", proj.Path)
		return
	}

	if _, err := os.Stat(artifacts); err == nil {
		fmt.Printf(" [%s] - build artifacts found in: \"%s\"...\n", proj.Path, artifacts)
	}

	processArtifacts(config.Home, artifacts, proj.Path, branchName)

}

func runProjectScripts(dir string, proj project) {
	fmt.Printf(" [%s] - project has %d scripts configured\n", proj.Path, len(proj.Scripts))

	for _, script := range proj.Scripts {
		fmt.Printf(" [%s] - executing project script: \"%s\"...\n", proj.Path, script)
		stdout, err := execInDir(dir, script)
		if err != nil {
			fmt.Printf(" [%s] - error executing project script: \"%s\"...\n", proj.Path, script)
			fmt.Printf("%s\n", string(stdout))
			panic(err)
		}

		//fmt.Printf("%s\n", string(stdout))
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
	fmt.Printf(" [%s] - processing build artifacts for project \"%s\", branch \"%s\".\n", project, project, branchName)

	destParent := home + "/artifacts/" + project
	destination := destParent + "/" + branchName

	fmt.Printf(" [%s] - build artifacts will be stored in: \"%s\".\n", project, destination)

	fmt.Printf(" [%s] - removing any previous artifacts from the destination\n", project)
	err := os.RemoveAll(destination)
	if err != nil {
		panic(err)
	}

	fmt.Printf(" [%s] - creating destination directory structure\n", project)
	err = os.MkdirAll(destParent, 0755)
	if err != nil {
		panic(err)
	}

	fmt.Printf(" [%s] - moving build artifacts into destination\n", project)
	err = os.Rename(artifacts, destination)
	if err != nil {
		panic(err)
	}

	fmt.Printf(" [%s] - artifact processing completed.\n", project)
}

func parseConfig(cfg string) *configuration {
	res := configuration{}
	json.Unmarshal([]byte(cfg), &res)
	fmt.Printf("Loaded Configuration: %d Projects Configured.\n", len(res.Projects))
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
	fmt.Printf(" [git] - running credentials callback with username \"%s\" for url \"%s\"\n", username, url)
	ret, cred := git.NewCredSshKeyFromAgent(username)
	return git.ErrorCode(ret), &cred
}

func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	fmt.Printf(" [git] - running certificate check callback for hostname \"%s\"\n", hostname)
	if hostname != "github.com" && hostname != "gitlab.com" {
		return git.ErrUser
	}
	return 0
}

func cloneRepo(twd string, url string, path string, cloneOpts *git.CloneOptions) (*git.Repository, error) {

	fmt.Printf(" [%s] - cloning repository from \"%s\" into \"%s\"\n", path, url, twd)

	// Clone
	repo, err := git.Clone(url, twd, cloneOpts)
	if err != nil {
		return nil, err
	}

	fmt.Printf(" [%s] - clone completed, finding head ref\n", path)

	// Get HEAD ref
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}

	fmt.Printf(" [%s] - looking up commit details for current head\n", path)

	// Find the commit at HEAD
	//headCommit, err := repo.LookupCommit(head.Target())
	//if err != nil {
	//	return nil, err
	//}

	fmt.Printf(" [%s] - head is now at %v\n", path, head.Target())

	return repo, nil
}

func checkoutBranch(repo *git.Repository, branchName string) error {
	checkoutOpts := &git.CheckoutOpts{
		Strategy: git.CheckoutSafe | git.CheckoutRecreateMissing | git.CheckoutAllowConflicts | git.CheckoutUseTheirs,
	}
	//Getting the reference for the remote branch
	// remoteBranch, err := repo.References.Lookup("refs/remotes/origin/" + branchName)
	remoteBranch, err := repo.LookupBranch("origin/"+branchName, git.BranchRemote)
	if err != nil {
		log.Print("Failed to find remote branch: " + branchName)
		return err
	}
	defer remoteBranch.Free()

	// Lookup for commit from remote branch
	commit, err := repo.LookupCommit(remoteBranch.Target())
	if err != nil {
		log.Print("Failed to find remote branch commit: " + branchName)
		return err
	}
	defer commit.Free()

	localBranch, err := repo.LookupBranch(branchName, git.BranchLocal)
	// No local branch, lets create one
	if localBranch == nil || err != nil {
		// Creating local branch
		localBranch, err = repo.CreateBranch(branchName, commit, false)
		if err != nil {
			log.Print("Failed to create local branch: " + branchName)
			return err
		}

		// Setting upstream to origin branch
		err = localBranch.SetUpstream("origin/" + branchName)
		if err != nil {
			log.Print("Failed to create upstream to origin/" + branchName)
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
		log.Print("Failed to lookup for commit in local branch " + branchName)
		return err
	}
	defer localCommit.Free()

	tree, err := repo.LookupTree(localCommit.TreeId())
	if err != nil {
		log.Print("Failed to lookup for tree " + branchName)
		return err
	}
	defer tree.Free()

	// Checkout the tree
	err = repo.CheckoutTree(tree, checkoutOpts)
	if err != nil {
		log.Print("Failed to checkout tree " + branchName)
		return err
	}
	// Setting the Head to point to our branch
	repo.SetHead("refs/heads/" + branchName)
	return nil
}

func printRepoDebug(repo *git.Repository) {
	odb, err := repo.Odb()
	if err != nil {
		log.Fatal(err)
	}

	err2 := odb.ForEach(func(oid *git.Oid) error {
		obj, err := repo.Lookup(oid)
		if err != nil {
			return err
		}

		//switch obj := obj.(type) {
		//default:
		//case *git.Blob:
		//	break
		fmt.Printf("==================================\n")
		fmt.Printf("obj:  %s\n", obj)
		fmt.Printf("Type: %v\n", obj.Type())
		fmt.Printf("Id:   %v\n", obj.Id())
		//fmt.Printf("Size: %s\n", obj.Size())
		//case *git.Commit:
		//	fmt.Printf("==================================\n")
		//	fmt.Printf("obj:  %s\n", obj)
		//	fmt.Printf("Type: %s\n", obj.Type())
		//	fmt.Printf("Id:   %s\n", obj.Id())
		//	author := obj.Author()
		//	fmt.Printf("    Author:\n        Name:  %s\n        Email: %s\n        Date:  %s\n", author.Name, author.Email, author.When)
		//	committer := obj.Committer()
		//	fmt.Printf("    Committer:\n        Name:  %s\n        Email: %s\n        Date:  %s\n", committer.Name, committer.Email, committer.When)
		//	fmt.Printf("    ParentCount: %s\n", int(obj.ParentCount()))
		//	fmt.Printf("    TreeId:      %s\n", obj.TreeId())
		//	fmt.Printf("    Message:\n\n        %s\n\n", strings.Replace(obj.Message(), "\n", "\n        ", -1))
		//fmt.Printf("obj.Parent: %s\n", obj.Parent())
		//fmt.Printf("obj.ParentId: %s\n", obj.ParentId())
		//fmt.Printf("obj.Tree: %s\n", obj.Tree())
		//case *git.Tree:
		//	break
		//	fmt.Printf("==================================\n")
		//	fmt.Printf("obj:  %s\n", obj)
		//	fmt.Printf("Type: %s\n", obj.Type())
		//	fmt.Printf("Id:   %s\n", obj.Id())
		//	fmt.Printf("    EntryCount: %s\n", obj.EntryCount())
		//}
		return nil
	})

	if err2 != nil {
		log.Fatal("Lookup:", err2)
	}
}
