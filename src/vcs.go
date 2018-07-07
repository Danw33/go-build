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

// vcs - Version Control: Git Interface
package main

import (
	"errors"
	"strings"
	"github.com/libgit2/git2go"
)

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

func fetchChanges(repo *git.Repository, fallbackURL string, project string) error {

	log.Debugf(" [%s] - Looking up remote \"origin\"...", project)

	remote, err := repo.Remotes.Lookup("origin")
	if err != nil {
		log.Debugf(" [%s] - Remote \"origin\" does not exist, setting it to the configured project URL...", project)
		remote, err = repo.Remotes.Create("origin", fallbackURL)
		if err != nil {
			return err
		}
	}

	// Fetch Options + Callbacks
	fopts := &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CredentialsCallback:      credentialsCallback,
			CertificateCheckCallback: certificateCheckCallback,
		},
		UpdateFetchhead: true,
	}

	log.Debugf(" [%s] - Fetching changes from remote \"origin\"...", project)
	err = remote.Fetch([]string{}, fopts, "")
	if err != nil {
		return err
	}

	return nil
}

func pullChanges(repo *git.Repository, project string) error {

	head, headErr := repo.Head()
	if headErr != nil {
		return nil
	}

	if head == nil {
		return errors.New("Failed to find current HEAD")
	}

	// Find the branch name
	branch := ""
	branchElements := strings.Split(head.Name(), "/")
	if len(branchElements) == 3 {
		branch = branchElements[2]
	}

	// Get remote ref for current branch
	remoteBranch, err := repo.References.Lookup("refs/remotes/origin/" + branch)
	if err != nil {
		return err
	}

	remoteBranchID := remoteBranch.Target()
	// Get annotated commit
	annotatedCommit, err := repo.AnnotatedCommitFromRef(remoteBranch)
	if err != nil {
		return err
	}

	// Do the merge analysis
	mergeHeads := make([]*git.AnnotatedCommit, 1)
	mergeHeads[0] = annotatedCommit
	analysis, _, err := repo.MergeAnalysis(mergeHeads)
	if err != nil {
		return err
	}

	if analysis&git.MergeAnalysisUpToDate != 0 {
		return nil
	} else if analysis&git.MergeAnalysisNormal != 0 {
		// Just merge changes
		if err := repo.Merge([]*git.AnnotatedCommit{annotatedCommit}, nil, nil); err != nil {
			return err
		}
		// Check for conflicts
		index, err := repo.Index()
		if err != nil {
			return err
		}

		if index.HasConflicts() {
			return errors.New("Conflicts encountered. Please resolve them")
		}

		// Make the merge commit
		sig, err := repo.DefaultSignature()
		if err != nil {
			return err
		}

		// Get Write Tree
		treeID, err := index.WriteTree()
		if err != nil {
			return err
		}

		tree, err := repo.LookupTree(treeID)
		if err != nil {
			return err
		}

		localCommit, err := repo.LookupCommit(head.Target())
		if err != nil {
			return err
		}

		remoteCommit, err := repo.LookupCommit(remoteBranchID)
		if err != nil {
			return err
		}

		repo.CreateCommit("HEAD", sig, sig, "", tree, localCommit, remoteCommit)

		// Clean up
		repo.StateCleanup()
	} else if analysis&git.MergeAnalysisFastForward != 0 {
		// Fast-forward changes
		// Get remote tree
		remoteTree, err := repo.LookupTree(remoteBranchID)
		if err != nil {
			return err
		}

		// Checkout
		if coErr := repo.CheckoutTree(remoteTree, nil); coErr != nil {
			return coErr
		}

		branchRef, err := repo.References.Lookup("refs/heads/" + branch)
		if err != nil {
			return err
		}

		// Point branch to the object
		branchRef.SetTarget(remoteBranchID, "")
		if _, err := head.SetTarget(remoteBranchID, ""); err != nil {
			return err
		}

	} else {
		log.Errorf(" [%s] - Unexpected merge analysis result %d", project, analysis)
		return errors.New("Unexpected merge analysis result")
	}

	return nil
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

func describeWorkDir(repo *git.Repository, project string) (string, error) {
	describeOpts, err := git.DefaultDescribeOptions()
	if err != nil {
		log.Error("Failed to load git describe options for project " + project)
		return "", err
	}

	formatOpts, err := git.DefaultDescribeFormatOptions()
	if err != nil {
		log.Error("Failed to load git describe format options for project " + project)
		return "", err
	}

	result, err := repo.DescribeWorkdir(&describeOpts)
	if err != nil {
		log.Error("Failed to describe working directory for project " + project)
		return "", err
	}

	resultStr, err := result.Format(&formatOpts)
	if err != nil {
		log.Error("Failed to format working directory description for project " + project)
		return "", err
	}

	return resultStr, nil
}
