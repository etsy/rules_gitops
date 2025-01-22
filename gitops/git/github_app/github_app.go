package github_app

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v68/github"
)

var (
	repoOwner               = flag.String("github_app_repo_owner", "", "the owner user/organization to use for github api requests")
	repo                    = flag.String("github_app_repo", "", "the repo to use for github api requests")
	githubEnterpriseHost    = flag.String("github_app_enterprise_host", "", "The host name of the private enterprise github, e.g. git.corp.adobe.com")
	message                 = flag.String("message", "", "Message to send")
	privateKey              = flag.String("private_key", "/var/run/agent-secrets/buildkite-agent/secrets/github-pr-creator-key", "Private Key")
	gitHubAppId             = flag.Int64("github_app_id", 1014336, "GitHub App Id")
	gitHubAppInstallationId = flag.Int64("github_installation_id", 0, "GitHub App Id")
	gitHubUser              = flag.String("github_user", "etsy", "GitHub User")
	gitHubAppName           = flag.String("github_app_name", "gitops-pr-creator", "Name of the GitHub App")
)

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

func CreateCommit(baseBranch string, commitBranch string, gitopsPath string, files []string, commitMsg string) {
	ctx := context.Background()
	gh := createGithubClient()

	log.Printf("Starting Create Commit: Commit branch: %s\n", commitBranch)
	log.Printf("Starting Create Commit: Base branch: %s\n", baseBranch)
	ref := getRef(ctx, gh, baseBranch, commitBranch)
	tree, err := getTree(ctx, gh, ref, gitopsPath, files)
	if err != nil {
		log.Fatalf("failed to create tree: %v", err)
	}

	pushCommit(ctx, gh, ref, tree, commitMsg)
	createPR(ctx, gh, baseBranch, commitBranch, commitMsg, "")
}

func createPR(ctx context.Context, gh *github.Client, baseBranch string, commitBranch string, prSubject string, prDescription string) {
	newPR := &github.NewPullRequest{
		Title:               &prSubject,
		Head:                &commitBranch,
		Base:                &baseBranch, // This is the target branch
		Body:                &prDescription,
		MaintainerCanModify: github.Bool(true),
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

func getTree(ctx context.Context, gh *github.Client, ref *github.Reference, gitopsPath string, filePaths []string) (tree *github.Tree, err error) {
	// Create a tree with what to commit.
	entries := []*github.TreeEntry{}

	// Load each file into the tree.
	for _, file := range filePaths {
		fullPath := filepath.Join(gitopsPath, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Fatalf("failed to read file %s: %v", fullPath, err)
		}

		entries = append(entries, &github.TreeEntry{Path: github.String(file), Type: github.String("blob"), Content: github.String(string(content)), Mode: github.String("100644")})
	}

	tree, _, err = gh.Git.CreateTree(ctx, *repoOwner, *repo, *ref.Object.SHA, entries)
	return tree, err
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
