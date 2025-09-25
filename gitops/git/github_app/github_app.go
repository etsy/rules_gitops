package github_app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v68/github"
)

var (
	repoOwner               = flag.String("github_app_repo_owner", "", "the owner user/organization to use for github api requests")
	repo                    = flag.String("github_app_repo", "", "the repo to use for github api requests")
	githubEnterpriseHost    = flag.String("github_app_enterprise_host", "", "The host name of the private enterprise github, e.g. git.corp.adobe.com")
	privateKey              = flag.String("private_key", "/var/run/agent-secrets/buildkite-agent/secrets/github-pr-creator-key", "Private Key")
	gitHubAppId             = flag.Int64("github_app_id", 1014336, "GitHub App Id")
	gitHubAppInstallationId = flag.Int64("github_installation_id", 0, "GitHub App Id")
)

type FileEntry struct {
	RelativePath string // Path for GitHub repository
	FullPath     string // Path for local file reading
}

func CreatePR(from, to, title, body string) error {
	if *repoOwner == "" {
		return errors.New("github_app_repo_owner must be set")
	}
	if *repo == "" {
		return errors.New("github_app_repo must be set")
	}
	if *gitHubAppId == 0 {
		return errors.New("github_app_id must be set")
	}

	ctx := context.Background()

	// get an installation token request handler for the github app
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, *gitHubAppId, *gitHubAppInstallationId, *privateKey)
	if err != nil {
		log.Println("failed reading key", "key", *privateKey, "err", err)
		return err
	}

	var gh *github.Client
	if *githubEnterpriseHost != "" {
		baseUrl := "https://" + *githubEnterpriseHost + "/api/v3/"
		uploadUrl := "https://" + *githubEnterpriseHost + "/api/uploads/"
		var err error
		gh, err = github.NewEnterpriseClient(baseUrl, uploadUrl, &http.Client{Transport: itr})
		if err != nil {
			log.Println("Error in creating github client", err)
			return nil
		}
	} else {
		gh = github.NewClient(&http.Client{Transport: itr})
	}

	pr := &github.NewPullRequest{
		Title:               &title,
		Head:                &from,
		Base:                &to,
		Body:                &body,
		Issue:               nil,
		MaintainerCanModify: new(bool),
		Draft:               new(bool),
	}
	createdPr, resp, err := gh.PullRequests.Create(ctx, *repoOwner, *repo, pr)
	if err == nil {
		log.Println("Created PR: ", *createdPr.URL)
		return err
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		// Handle the case: "Create PR" request fails because it already exists
		log.Println("Reusing existing PR")
		return nil
	}

	// All other github responses
	defer resp.Body.Close()
	responseBody, readingErr := io.ReadAll(resp.Body)
	if readingErr != nil {
		log.Println("cannot read response body")
	} else {
		log.Println("github response: ", string(responseBody))
	}

	return err
}

func CreateCommit(baseBranch string, commitBranch string, gitopsPath string, files []string, deletedFiles []string, prTitle string, prDescription string, branchesNeedingRecreation []string) {
	ctx := context.Background()
	gh := createGithubClient()

	log.Printf("Starting Create Commit: Commit branch: %s\n", commitBranch)
	log.Printf("Starting Create Commit: Base branch: %s\n", baseBranch)
	log.Printf("GitOps Path: %s\n", gitopsPath)
	log.Printf("Modified Files: %v\n", files)
	log.Printf("Deleted Files: %v\n", deletedFiles)
	log.Printf("Branches needing recreation: %v\n", branchesNeedingRecreation)

	// Check if this branch needs recreation due to deletions
	needsRecreation := false
	branchName := fmt.Sprintf("deploy/%s", commitBranch)
	for _, recreateBranch := range branchesNeedingRecreation {
		if recreateBranch == branchName {
			needsRecreation = true
			break
		}
	}

	var ref *github.Reference
	var err error

	if needsRecreation {
		log.Printf("Branch %s needs recreation due to target deletions, force-resetting from %s\n", branchName, baseBranch)
		ref, err = forceResetBranch(ctx, gh, baseBranch, branchName)
		if err != nil {
			log.Fatalf("failed to force reset branch: %v", err)
		}

		// When recreating, we need to include all current files in the GitOps directory
		allFileEntries, err := getAllFilesToCommit(gitopsPath)
		if err != nil {
			log.Fatalf("failed to get all files to commit: %v", err)
		}

		tree, err := getTree(ctx, gh, ref, allFileEntries)
		if err != nil {
			log.Fatalf("failed to create tree: %v", err)
		}

		pushCommit(ctx, gh, ref, tree, prTitle)
	} else {
		// Handle incremental changes (adds/modifies/deletes)
		ref = getRef(ctx, gh, baseBranch, branchName)

		tree, err := getTreeWithChanges(ctx, gh, ref, gitopsPath, files, deletedFiles)
		if err != nil {
			log.Fatalf("failed to create tree with changes: %v", err)
		}

		pushCommit(ctx, gh, ref, tree, prTitle)
	}

	createPR(ctx, gh, baseBranch, branchName, prTitle, prDescription)
}

func getFilesToCommit(gitopsPath string, inputPaths []string) ([]FileEntry, error) {
	var allFileEntries []FileEntry

	for _, inputPath := range inputPaths {
		inputPath = strings.TrimSuffix(inputPath, "/")
		absInputPath := filepath.Join(gitopsPath, inputPath)

		info, err := os.Stat(absInputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to access input path %s: %v", absInputPath, err)
		}

		if info.IsDir() {
			err = filepath.Walk(absInputPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					// Get path relative to gitopsPath
					relPath, err := filepath.Rel(gitopsPath, path)
					if err != nil {
						return fmt.Errorf("failed to get relative path for %s: %v", path, err)
					}
					allFileEntries = append(allFileEntries, FileEntry{
						RelativePath: relPath,
						FullPath:     path,
					})
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk input directory: %v", err)
			}
		} else {
			// For single files, use the inputPath directly as it's already relative to gitopsPath
			allFileEntries = append(allFileEntries, FileEntry{
				RelativePath: inputPath,
				FullPath:     absInputPath,
			})
		}
	}

	if len(allFileEntries) == 0 {
		return nil, fmt.Errorf("no files found in input paths %v", inputPaths)
	}

	return allFileEntries, nil
}

func createPR(ctx context.Context, gh *github.Client, baseBranch string, commitBranch string, prSubject string, prDescription string) {
	newPR := &github.NewPullRequest{
		Title:               &prSubject,
		Head:                &commitBranch,
		Base:                &baseBranch, // This is the target branch
		Body:                &prDescription,
		MaintainerCanModify: github.Ptr(true),
	}

	pr, _, err := gh.PullRequests.Create(ctx, *repoOwner, *repo, newPR)
	if err != nil {
		log.Fatalf("failed to create PR: %v", err)
	}

	log.Printf("PR created: %s\n", pr.GetHTMLURL())
}

func createGithubClient() *github.Client {
	if *repoOwner == "" {
		log.Fatal("github_app_repo_owner must be set")
	}
	if *repo == "" {
		log.Fatal("github_app_repo must be set")
	}
	if *gitHubAppId == 0 {
		log.Fatal("github_app_id must be set")
	}

	// get an installation token request handler for the github app
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, *gitHubAppId, *gitHubAppInstallationId, *privateKey)
	if err != nil {
		log.Println("failed reading key", "key", *privateKey, "err", err)
		log.Fatal(err)
	}

	return github.NewClient(&http.Client{Transport: itr})
}

func getRef(ctx context.Context, gh *github.Client, baseBranch string, commitBranch string) *github.Reference {
	log.Printf("Creating ref for branch %s from %s\n", commitBranch, baseBranch)
	log.Printf("Getting ref for branch %s\n", baseBranch)
	baseRef, _, err := gh.Git.GetRef(ctx, *repoOwner, *repo, "refs/heads/"+baseBranch)

	if err != nil {
		log.Fatalf("failed to get base branch ref: %v", err)
	}

	log.Printf("Creating ref for branch %s\n", commitBranch)
	newRef := &github.Reference{Ref: github.String("refs/heads/" + commitBranch), Object: &github.GitObject{SHA: baseRef.Object.SHA}}

	ref, _, err := gh.Git.CreateRef(ctx, *repoOwner, *repo, newRef)
	if err != nil {
		log.Fatalf("failed to create branch ref: %v", err)
	}
	return ref
}

func getTreeWithChanges(ctx context.Context, gh *github.Client, ref *github.Reference, gitopsPath string, addedModifiedFiles []string, deletedFiles []string) (tree *github.Tree, err error) {
	entries := []*github.TreeEntry{}

	// Add/modify files
	if len(addedModifiedFiles) > 0 {
		fileEntries, err := getFilesToCommit(gitopsPath, addedModifiedFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to get files to commit: %v", err)
		}

		for _, file := range fileEntries {
			content, err := os.ReadFile(file.FullPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %v", file.FullPath, err)
			}
			log.Printf("Adding/modifying file %s to tree\n", file.RelativePath)
			entries = append(entries, &github.TreeEntry{
				Path:    github.Ptr(file.RelativePath),
				Type:    github.Ptr("blob"),
				Content: github.Ptr(string(content)),
				Mode:    github.Ptr("100644"),
			})
		}
	}

	// Delete files by setting SHA to nil
	for _, deletedFile := range deletedFiles {
		log.Printf("Deleting file %s from tree\n", deletedFile)
		entries = append(entries, &github.TreeEntry{
			Path: github.Ptr(deletedFile),
			Mode: github.Ptr("100644"),
			Type: github.Ptr("blob"),
			SHA:  nil, // Setting SHA to nil deletes the file
		})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no changes to commit")
	}

	tree, _, err = gh.Git.CreateTree(ctx, *repoOwner, *repo, *ref.Object.SHA, entries)
	return tree, err
}

func getTree(ctx context.Context, gh *github.Client, ref *github.Reference, files []FileEntry) (tree *github.Tree, err error) {
	// Create a tree with what to commit.
	entries := []*github.TreeEntry{}

	// Load each file into the tree.
	for _, file := range files {
		content, err := os.ReadFile(file.FullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %v", file.FullPath, err)
		}
		log.Printf("Adding file %s to tree\n", file.RelativePath)
		entries = append(entries, &github.TreeEntry{
			Path:    github.Ptr(file.RelativePath),
			Type:    github.Ptr("blob"),
			Content: github.Ptr(string(content)),
			Mode:    github.Ptr("100644"),
		})
	}

	tree, _, err = gh.Git.CreateTree(ctx, *repoOwner, *repo, *ref.Object.SHA, entries)
	return tree, err
}

func forceResetBranch(ctx context.Context, gh *github.Client, baseBranch string, targetBranch string) (*github.Reference, error) {
	// Get the base branch reference
	baseRef, _, err := gh.Git.GetRef(ctx, *repoOwner, *repo, "refs/heads/"+baseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get base branch ref: %v", err)
	}

	// Try to get the existing target branch
	targetRef, _, err := gh.Git.GetRef(ctx, *repoOwner, *repo, "refs/heads/"+targetBranch)
	if err != nil {
		// Branch doesn't exist, create it
		log.Printf("Target branch %s doesn't exist, creating it\n", targetBranch)
		newRef := &github.Reference{
			Ref:    github.String("refs/heads/" + targetBranch),
			Object: &github.GitObject{SHA: baseRef.Object.SHA},
		}
		createdRef, _, err := gh.Git.CreateRef(ctx, *repoOwner, *repo, newRef)
		if err != nil {
			return nil, fmt.Errorf("failed to create branch ref: %v", err)
		}
		return createdRef, nil
	}

	// Branch exists, force update it to point to base branch
	log.Printf("Force updating branch %s to match %s\n", targetBranch, baseBranch)
	targetRef.Object.SHA = baseRef.Object.SHA
	updatedRef, _, err := gh.Git.UpdateRef(ctx, *repoOwner, *repo, targetRef, true) // force=true
	if err != nil {
		return nil, fmt.Errorf("failed to force update branch ref: %v", err)
	}

	return updatedRef, nil
}

func getAllFilesToCommit(gitopsPath string) ([]FileEntry, error) {
	var allFileEntries []FileEntry

	// Walk through the entire GitOps directory and get all files
	err := filepath.Walk(gitopsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Get path relative to gitopsPath
			relPath, err := filepath.Rel(gitopsPath, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path for %s: %v", path, err)
			}
			// Skip hidden files and git files
			if !strings.HasPrefix(filepath.Base(relPath), ".") {
				allFileEntries = append(allFileEntries, FileEntry{
					RelativePath: relPath,
					FullPath:     path,
				})
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk GitOps directory: %v", err)
	}

	if len(allFileEntries) == 0 {
		return nil, fmt.Errorf("no files found in GitOps directory %s", gitopsPath)
	}

	log.Printf("Found %d files in GitOps directory\n", len(allFileEntries))
	return allFileEntries, nil
}

func pushCommit(ctx context.Context, gh *github.Client, ref *github.Reference, tree *github.Tree, commitMessage string) {
	// Get the parent commit to attach the commit to.
	parent, _, err := gh.Repositories.GetCommit(ctx, *repoOwner, *repo, *ref.Object.SHA, nil)
	if err != nil {
		log.Fatalf("failed to get parent commit: %v", err)
	}
	// This is not always populated, but is needed.
	parent.Commit.SHA = parent.SHA

	// Create the commit using the tree.
	commit := &github.Commit{Message: &commitMessage, Tree: tree, Parents: []*github.Commit{parent.Commit}}
	opts := github.CreateCommitOptions{}

	newCommit, _, err := gh.Git.CreateCommit(ctx, *repoOwner, *repo, commit, &opts)
	if err != nil {
		log.Fatalf("failed to create commit: %v", err)
	}

	// Attach the commit to the master branch.
	ref.Object.SHA = newCommit.SHA
	_, _, err = gh.Git.UpdateRef(ctx, *repoOwner, *repo, ref, false)
	if err != nil {
		log.Fatalf("failed to update ref: %v", err)
	}
}
