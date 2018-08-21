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
	"github.com/getsentry/raven-go"
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
	Log.Debugf(" [git] - running credentials callback with username \"%s\" for url \"%s\"\n", username, url)
	ret, cred := git.NewCredSshKeyFromAgent(username)
	return git.ErrorCode(ret), &cred
}

func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	Log.Debugf(" [git] - running certificate check callback for hostname \"%s\"\n", hostname)
	if hostname != "github.com" && hostname != "gitlab.com" {
		Log.Debugf(" [git] - certificate check callback passed for hostname \"%s\"\n", hostname)
		return git.ErrUser
	}
	Log.Warningf(" [git] - certificate check callback passed for hostname \"%s\"\n", hostname)
	return 0
}

func cloneRepo(twd string, url string, path string, cloneOpts *git.CloneOptions) (*git.Repository, error) {

	Log.Debugf(" [%s] - cloning repository from \"%s\" into \"%s\"\n", path, url, twd)

	// Clone
	repo, err := git.Clone(url, twd, cloneOpts)
	if err != nil {
		raven.CaptureError(err, nil)
		return nil, err
	}

	Log.Debugf(" [%s] - clone completed, finding head ref\n", path)

	// Get HEAD ref
	head, err := repo.Head()
	if err != nil {
		raven.CaptureError(err, nil)
		return nil, err
	}

	Log.Debugf(" [%s] - head is now at %v\n", path, head.Target())

	return repo, nil
}

func fetchChanges(repo *git.Repository, fallbackURL string, project string) error {

	Log.Debugf(" [%s] - Looking up remote \"origin\"...", project)

	remote, err := repo.Remotes.Lookup("origin")
	if err != nil {
		Log.Debugf(" [%s] - Remote \"origin\" does not exist, setting it to the configured project URL...", project)
		remote, err = repo.Remotes.Create("origin", fallbackURL)
		if err != nil {
			raven.CaptureError(err, nil)
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

	Log.Debugf(" [%s] - Fetching changes from remote \"origin\"...", project)
	err = remote.Fetch([]string{}, fopts, "")
	if err != nil {
		raven.CaptureError(err, nil)
		return err
	}

	return nil
}

func pullChanges(repo *git.Repository, project string) error {

	head, headErr := repo.Head()
	if headErr != nil {
		Log.Errorf(" [%s] - Error whilst finding current HEAD for repository!", project)
		return headErr
	}

	if head == nil {
		Log.Errorf(" [%s] - Failed to find current HEAD for repository!", project)
		return errors.New("failed to find current HEAD")
	}

	// Find the branch name
	branch := ""
	hName := head.Name()
	Log.Debugf(" [%s] - Parsing head name '%s' to determine branch name", project, hName)
	branchElements := strings.Split(hName, "/")
	bECount := len(branchElements)
	if bECount == 3 {
		branch = branchElements[2]
	} else if len(branchElements) > 3 {
		branch = strings.Join(branchElements[2:], "/")
	} else {
		// Less than 3 or no count
		Log.Errorf(" [%s] - Failed to determine branch name from repository head!", project)
		return errors.New("invalid quantity of branch elements received for parsing")
	}
	Log.Debugf(" [%s] - Got branch name '%s' for head", project, branch)

	// Get remote ref for current branch
	remoteBranch, err := repo.References.Lookup("refs/remotes/origin/" + branch)
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Errorf(" [%s] - Failed to get remote ref for branch '%s' when using 'refs/remotes/origin/%s' for lookup", project, branch, branch)
		return err
	}
	Log.Debugf(" [%s] - Got remote ref for branch '%s' using 'refs/remotes/origin/%s'", project, branch, branch)

	remoteBranchID := remoteBranch.Target()
	// Get annotated commit
	annotatedCommit, err := repo.AnnotatedCommitFromRef(remoteBranch)
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Errorf(" [%s] - Failed to get annotated commit from remote branch ref '%s'!", project, remoteBranch)
		return err
	}
	Log.Debugf(" [%s] - Got annotated commit from remote ref", project)

	// Do the merge analysis
	Log.Debugf(" [%s] - Performing merge analysis...", project)
	mergeHeads := make([]*git.AnnotatedCommit, 1)
	mergeHeads[0] = annotatedCommit
	analysis, _, err := repo.MergeAnalysis(mergeHeads)
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Errorf(" [%s] - Failed to perform merge analysis!", project)
		return err
	}

	if analysis&git.MergeAnalysisUpToDate != 0 {
		Log.Debugf(" [%s] - Merge Analysis: Up-to-date!", project)
		return nil
	} else if analysis&git.MergeAnalysisNormal != 0 {
		Log.Debugf(" [%s] - Merge Analysis: Not Normal", project)

		// Just merge changes
		if err := repo.Merge([]*git.AnnotatedCommit{annotatedCommit}, nil, nil); err != nil {
			raven.CaptureError(err, nil)
			return err
		}
		// Check for conflicts
		index, err := repo.Index()
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf(" [%s] - Failed to determine the repository index!", project)
			return err
		}
		Log.Debugf(" [%s] - Repository index acquired", project)

		if index.HasConflicts() {
			Log.Errorf(" [%s] - Merge analysis found conflicts!", project)
			return errors.New("conflicts encountered. Please resolve them")
		}
		Log.Debugf(" [%s] - Merge analysis did not find any conflicts.", project)

		// Make the merge commit
		sig, err := repo.DefaultSignature()
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf(" [%s] - Error performing merge commit using default signature", project)
			return err
		}
		Log.Debugf(" [%s] - Merge commit completed using default signature", project)

		// Get Write Tree
		treeID, err := index.WriteTree()
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf(" [%s] - Failed to get the write tree from the current index!", project)
			return err
		}
		Log.Debugf(" [%s] - Write Tree ID acquired from index", project)

		tree, err := repo.LookupTree(treeID)
		if err != nil {
			raven.CaptureError(err, nil)
			return err
		}
		Log.Debugf(" [%s] - Tree lookup completed based on write tree ID", project)

		localCommit, err := repo.LookupCommit(head.Target())
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf(" [%s] - Failed to lookup local commit from head target!", project)
			return err
		}
		Log.Debugf(" [%s] - Local commit for head target found", project)

		remoteCommit, err := repo.LookupCommit(remoteBranchID)
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf(" [%s] - Failed to lookup remote commit from remote branch ID '%s'!", project, remoteBranchID)
			return err
		}
		Log.Debugf(" [%s] - Remote commit for remote branch ID '%s' found", project, remoteBranchID)

		repo.CreateCommit("HEAD", sig, sig, "", tree, localCommit, remoteCommit)

		// Clean up
		Log.Debugf(" [%s] - Performing state cleanup post-commit", project)
		repo.StateCleanup()
	} else if analysis&git.MergeAnalysisFastForward != 0 {
		Log.Debugf(" [%s] - Merge Analysis: Fast-Forward", project)

		// Fast-forward changes
		// Get remote tree
		remoteTree, err := repo.LookupTree(remoteBranchID)
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf(" [%s] - Failed to lookup remote tree for remote branch ID '%s' during fast-forward!", project, remoteBranchID)
			return err
		}
		Log.Debugf(" [%s] - Found remote tree for remote branch ID '%s'", project, remoteBranchID)

		// Checkout
		if coErr := repo.CheckoutTree(remoteTree, nil); coErr != nil {
			raven.CaptureError(coErr, nil)
			Log.Errorf(" [%s] - Failed to checkout remote tree during fast-forward!", project)
			return coErr
		}
		Log.Debugf(" [%s] - Checked out remote tree", project)

		branchRef, err := repo.References.Lookup("refs/heads/" + branch)
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf(" [%s] - Failed to lookup branch ref for '%s' as 'refs/heads/%s' during fast-forward!", project, branch, branch)
			return err
		}
		Log.Debugf(" [%s] - Looked up branch ref for '%s' as 'refs/heads/%s'.", project, branch, branch)

		// Point branch to the object
		branchRef.SetTarget(remoteBranchID, "")
		if _, err := head.SetTarget(remoteBranchID, ""); err != nil {
			raven.CaptureError(err, nil)
			Log.Errorf(" [%s] - Failed to set branch ref to target object ID; Fast forward failed!", project)
			return err
		}
		Log.Debugf(" [%s] - Set branch ref to target object ID, Fast-forward complete.", project)

	} else {
		Log.Errorf(" [%s] - Unexpected merge analysis result %d", project, analysis)
		return errors.New("unexpected merge analysis result")
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
		raven.CaptureError(err, nil)
		Log.Error("Failed to find remote branch: " + branchName)
		return err
	}
	defer remoteBranch.Free()

	// Lookup for commit from remote branch
	commit, err := repo.LookupCommit(remoteBranch.Target())
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Error("Failed to find remote branch commit: " + branchName)
		return err
	}
	defer commit.Free()

	localBranch, err := repo.LookupBranch(branchName, git.BranchLocal)
	// No local branch, lets create one
	if localBranch == nil || err != nil {
		// Creating local branch
		localBranch, err = repo.CreateBranch(branchName, commit, false)
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Error("Failed to create local branch: " + branchName)
			return err
		}

		// Setting upstream to origin branch
		err = localBranch.SetUpstream("origin/" + branchName)
		if err != nil {
			raven.CaptureError(err, nil)
			Log.Error("Failed to create upstream to origin/" + branchName)
			return err
		}
	}
	if localBranch == nil {
		return errors.New("error while locating/creating local branch")
	}
	defer localBranch.Free()

	// Getting the tree for the branch
	localCommit, err := repo.LookupCommit(localBranch.Target())
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Error("Failed to lookup for commit in local branch " + branchName)
		return err
	}
	defer localCommit.Free()

	tree, err := repo.LookupTree(localCommit.TreeId())
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Error("Failed to lookup for tree " + branchName)
		return err
	}
	defer tree.Free()

	// Checkout the tree
	err = repo.CheckoutTree(tree, checkoutOpts)
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Error("Failed to checkout tree " + branchName)
		return err
	}

	// Setting the Head to point to our branch
	repo.SetHead("refs/heads/" + branchName)
	return nil
}

func describeWorkDir(repo *git.Repository, project string) (string, error) {
	describeOpts, err := git.DefaultDescribeOptions()
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Error("Failed to load git describe options for project " + project)
		return "", err
	}

	formatOpts, err := git.DefaultDescribeFormatOptions()
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Error("Failed to load git describe format options for project " + project)
		return "", err
	}

	result, err := repo.DescribeWorkdir(&describeOpts)
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Error("Failed to describe working directory for project " + project)
		return "", err
	}

	resultStr, err := result.Format(&formatOpts)
	if err != nil {
		raven.CaptureError(err, nil)
		Log.Error("Failed to format working directory description for project " + project)
		return "", err
	}

	return resultStr, nil
}
