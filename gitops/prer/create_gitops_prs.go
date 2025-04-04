/*
Copyright 2020 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	osexec "os/exec"
	"strings"
	"sync"

	"github.com/fasterci/rules_gitops/gitops/analysis"
	"github.com/fasterci/rules_gitops/gitops/bazel"
	"github.com/fasterci/rules_gitops/gitops/commitmsg"
	"github.com/fasterci/rules_gitops/gitops/exec"
	"github.com/fasterci/rules_gitops/gitops/git"
	"github.com/fasterci/rules_gitops/gitops/git/bitbucket"
	"github.com/fasterci/rules_gitops/gitops/git/github"
	"github.com/fasterci/rules_gitops/gitops/git/github_app"
	"github.com/fasterci/rules_gitops/gitops/git/gitlab"
	proto "github.com/golang/protobuf/proto"
)

// Config holds all command line configuration
type Config struct {
	// Git related configs
	GitRepo        string
	GitMirror      string
	GitHost        string
	BranchName     string
	GitCommit      string
	ReleaseBranch  string
	PRTargetBranch string

	// Bazel related configs
	BazelCmd  string
	Workspace string
	Targets   string

	// GitOps related configs
	GitOpsPath      string
	GitOpsTmpDir    string
	PushParallelism int
	DryRun          bool

	// PR related configs
	PRTitle                string
	PRBody                 string
	DeploymentBranchSuffix string

	// create_gitops_prs rule
	ResolvedBinaries SliceFlags
	ResolvedPushes   SliceFlags

	// Dependencies
	DependencyKinds []string
	DependencyNames []string
	DependencyAttrs []string
}

func initConfig() *Config {
	cfg := &Config{}

	// Git flags
	flag.StringVar(&cfg.GitRepo, "git_repo", "", "Git repository location")
	flag.StringVar(&cfg.GitMirror, "git_mirror", "", "Git mirror location (e.g., /mnt/mirror/repo.git)")
	flag.StringVar(&cfg.GitHost, "git_server", "bitbucket", "Git server API to use: 'bitbucket', 'github', 'gitlab', or 'github_app'")
	flag.StringVar(&cfg.BranchName, "branch_name", "unknown", "Branch name for commit message")
	flag.StringVar(&cfg.GitCommit, "git_commit", "unknown", "Git commit for commit message")
	flag.StringVar(&cfg.ReleaseBranch, "release_branch", "master", "Filter GitOps targets by release branch")
	flag.StringVar(&cfg.PRTargetBranch, "gitops_pr_into", "master", "Target branch for deployment PR")

	// Bazel flags
	flag.StringVar(&cfg.BazelCmd, "bazel_cmd", "tools/bazel", "Bazel binary path")
	flag.StringVar(&cfg.Workspace, "workspace", "", "Workspace root path")
	flag.StringVar(&cfg.Targets, "targets", "//... except //experimental/...", "Targets to scan (separate multiple with +)")

	// GitOps flags
	flag.StringVar(&cfg.GitOpsPath, "gitops_path", "cloud", "File storage location in repo")
	flag.StringVar(&cfg.GitOpsTmpDir, "gitops_tmpdir", os.TempDir(), "Git tree checkout location")
	flag.IntVar(&cfg.PushParallelism, "push_parallelism", 1, "Concurrent image push count")
	flag.BoolVar(&cfg.DryRun, "dry_run", false, "Print actions without creating PRs")

	// PR flags
	flag.StringVar(&cfg.PRTitle, "gitops_pr_title", "", "PR title")
	flag.StringVar(&cfg.PRBody, "gitops_pr_body", "", "PR body message")
	flag.StringVar(&cfg.DeploymentBranchSuffix, "deployment_branch_suffix", "", "Suffix for deployment branch names")

	// create_gitops_prs rule sets these when used with `bazel run`
	flag.Var(&cfg.ResolvedBinaries, "resolved_binary", "list of resolved gitops binaries to run. Can be specified multiple times. format is releasetrain:cmd/binary/to/run/command. Default is empty")
	flag.Var(&cfg.ResolvedPushes, "resolved_push", "list of resolved push binaries to run. Can be specified multiple times. format is cmd/binary/to/run/command. Default is empty")

	// Dependencies
	var kinds, names, attrs SliceFlags
	flag.Var(&kinds, "gitops_dependencies_kind", "Dependency kinds for GitOps phase")
	flag.Var(&names, "gitops_dependencies_name", "Dependency names for GitOps phase")
	flag.Var(&attrs, "gitops_dependencies_attr", "Dependency attributes (format: attr=value)")

	flag.Parse()

	cfg.DependencyKinds = kinds
	if len(cfg.DependencyKinds) == 0 {
		cfg.DependencyKinds = []string{"k8s_container_push", "push_oci"}
	}
	cfg.DependencyNames = names
	cfg.DependencyAttrs = attrs

	return cfg
}

func getGitServer(host string) git.Server {
	servers := map[string]git.Server{
		"github":     git.ServerFunc(github.CreatePR),
		"gitlab":     git.ServerFunc(gitlab.CreatePR),
		"bitbucket":  git.ServerFunc(bitbucket.CreatePR),
		"github_app": git.ServerFunc(github_app.CreatePR),
	}

	server, exists := servers[host]
	if !exists {
		log.Fatalf("unsupported git host: %s", host)
	}
	return server
}

func executeBazelQuery(query string) *analysis.CqueryResult {
	log.Printf("Running Bazel Query: %s", query)
	cmd := osexec.Command("bazel", "cquery",
		"--output=proto",
		"--noimplicit_deps",
		query)

	output, err := cmd.Output()
	if err != nil {
		log.Fatal("no protobuf data found in output")
	}

	result := &analysis.CqueryResult{}
	if err := proto.Unmarshal(output, result); err != nil {
		log.Fatalf("failed to unmarshal protobuf: %v", err)
	}

	return result
}

func processResolvedImages(cfg *Config) {
	resolvedPushChan := make(chan string)
	var wg sync.WaitGroup
	wg.Add(cfg.PushParallelism)

	for i := 0; i < cfg.PushParallelism; i++ {
		go func() {
			defer wg.Done()
			for cmd := range resolvedPushChan {
				exec.Mustex("", cmd)
			}
		}()
	}

	for _, r := range cfg.ResolvedPushes {
		resolvedPushChan <- r
	}
	close(resolvedPushChan)
	wg.Wait()
}

func processImages(targets []string, cfg *Config) {
	deps := fmt.Sprintf("set('%s')", strings.Join(targets, "' '"))
	queries := []string{}

	// Build queries
	for _, kind := range cfg.DependencyKinds {
		queries = append(queries, fmt.Sprintf("kind(%s, deps(%s))", kind, deps))
	}
	for _, name := range cfg.DependencyNames {
		queries = append(queries, fmt.Sprintf("filter(%s, deps(%s))", name, deps))
	}
	for _, attr := range cfg.DependencyAttrs {
		name, value, _ := strings.Cut(attr, "=")
		if value == "" {
			value = ".*"
		}
		queries = append(queries, fmt.Sprintf("attr(%s, %s, deps(%s))", name, value, deps))
	}

	query := strings.Join(queries, " union ")
	result := executeBazelQuery(query)

	// Process targets in parallel
	targetChan := make(chan string)
	var wg sync.WaitGroup
	wg.Add(cfg.PushParallelism)

	for i := 0; i < cfg.PushParallelism; i++ {
		go func() {
			defer wg.Done()
			for target := range targetChan {
				processTarget(target, cfg.BazelCmd)
			}
		}()
	}

	for _, t := range result.Results {
		targetChan <- t.Target.Rule.GetName()
	}
	close(targetChan)
	wg.Wait()
}

func processTarget(target, bazelCmd string) {
	executable := bazel.TargetToExecutable(target)
	if fi, err := os.Stat(executable); err == nil && fi.Mode().IsRegular() {
		exec.Mustex("", executable)
		return
	}
	log.Printf("target %s is not a file, running as command", target)
	exec.Mustex("", bazelCmd, "run", target)
}

func createPullRequests(branches []string, cfg *Config) {
	if cfg.DryRun {
		log.Printf("Dry run: would create PRs for branches: %v", branches)
		return
	}

	server := getGitServer(cfg.GitHost)
	for _, branch := range branches {
		title := cfg.PRTitle
		if title == "" {
			title = fmt.Sprintf("GitOps deployment %s", branch)
		}

		body := cfg.PRBody
		if body == "" {
			body = branch
		}

		if err := server.CreatePR(branch, cfg.PRTargetBranch, title, body); err != nil {
			log.Fatalf("failed to create PR: %v", err)
		}
	}
}

func main() {
	cfg := initConfig()

	if cfg.Workspace != "" {
		if err := os.Chdir(cfg.Workspace); err != nil {
			log.Fatalf("failed to change directory: %v", err)
		}
	}

	trains := make(map[string][]string)
	if len(cfg.ResolvedBinaries) > 0 {
		// This condition is used when calling the script from create_gitops_pr rules
		// When you call `bazel run <create_gitops_pr target>`, you can't call another bazel query within a bazel run command
		// So we have to rely on resolved binaries that were passed in
		for _, rb := range cfg.ResolvedBinaries {
			releaseTrain, bin, found := strings.Cut(rb, ":")
			if !found {
				log.Fatalf("resolved_binaries: invalid resolved_binary format: %s", rb)
			}
			trains[releaseTrain] = append(trains[releaseTrain], bin)
		}
	} else {
		// Find release trains
		query := fmt.Sprintf("attr(deployment_branch, \".+\", attr(release_branch_prefix, \"%s\", kind(gitops, %s)))",
			cfg.ReleaseBranch, cfg.Targets)

		result := executeBazelQuery(query)

		for _, t := range result.Results {
			for _, attr := range t.Target.GetRule().GetAttribute() {
				if attr.GetName() == "deployment_branch" {
					trains[attr.GetStringValue()] = append(trains[attr.GetStringValue()], t.Target.Rule.GetName())
				}
			}
		}
	}

	if len(trains) == 0 {
		log.Println("No matching targets found")
		return
	}

	// Create temporary directory
	gitopsDir, err := os.MkdirTemp(cfg.GitOpsTmpDir, "gitops")
	if err != nil {
		log.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(gitopsDir)

	// Clone repository
	workdir, err := git.Clone(cfg.GitRepo, gitopsDir, cfg.GitMirror, cfg.PRTargetBranch, cfg.GitOpsPath)
	if err != nil {
		log.Fatalf("failed to clone repository: %v", err)
	}

	var updatedTargets []string
	var updatedBranches []string
	var modifiedFiles []string

	// Process each release train
	for train, targets := range trains {
		branch := fmt.Sprintf("deploy/%s%s", train, cfg.DeploymentBranchSuffix)

		if !workdir.SwitchToBranch(branch, cfg.PRTargetBranch) {
			// Check if branch needs recreation due to deleted targets
			msg := workdir.GetLastCommitMessage()
			currentTargets := make(map[string]bool)
			for _, t := range targets {
				currentTargets[t] = true
			}

			for _, t := range commitmsg.ExtractTargets(msg) {
				if !currentTargets[t] {
					workdir.RecreateBranch(branch, cfg.PRTargetBranch)
					break
				}
			}
		}

		// Process targets
		for _, target := range targets {
			bin := bazel.TargetToExecutable(target)
			exec.Mustex("", bin, "--nopush", "--deployment_root", gitopsDir)
		}

		commitMsg := fmt.Sprintf("GitOps for release branch %s from %s commit %s\n%s",
			cfg.ReleaseBranch, cfg.BranchName, cfg.GitCommit, commitmsg.Generate(targets))

		files, err := workdir.GetModifiedFiles()

		if err != nil {
			log.Fatalf("failed to get modified files: %v", err)
		}

		modifiedFiles = append(modifiedFiles, files...)
		log.Printf("Modified files: %v", modifiedFiles)
		if workdir.Commit(commitMsg, cfg.GitOpsPath) {
			log.Printf("Branch %s has changes, push required", branch)
			updatedTargets = append(updatedTargets, targets...)
			updatedBranches = append(updatedBranches, branch)
		}
	}

	if len(updatedTargets) == 0 {
		log.Println("No GitOps changes to push")
		return
	}

	if len(cfg.ResolvedPushes) > 0 {
		processResolvedImages(cfg)
	} else {
		processImages(updatedTargets, cfg)
	}

	if !cfg.DryRun {
		slug := os.Getenv("BUILDKITE_PIPELINE_SLUG")
		url := os.Getenv("BUILDKITE_BUILD_URL")
		repo := os.Getenv("BUILDKITE_REPO")
		sha := os.Getenv("BUILDKITE_COMMIT")
		repo = strings.Replace(repo, ":", "/", 1)
		repo = strings.Replace(repo, "git@", "https://", 1)
		repo = strings.Replace(repo, ".git", "", 1)
		commit := fmt.Sprintf("%s/commit/%s", repo, sha)
		shortSha := sha[:7]

		prTitle := fmt.Sprintf("Gitops Deploy: %s - %s", slug, shortSha)
		prDescription := fmt.Sprintf("Automated PR for [%s](%s) via [Buildkite Pipeline](%s)", slug, commit, url)

		switch cfg.GitHost {
		case "github_app":
			github_app.CreateCommit(cfg.PRTargetBranch, cfg.BranchName, gitopsDir, modifiedFiles, prTitle, prDescription)
			return
		default:
			workdir.Push(updatedBranches)
			createPullRequests(updatedBranches, cfg)
		}
	}
}

// SliceFlags implements flag.Value for string slice flags
type SliceFlags []string

func (sf *SliceFlags) String() string {
	return fmt.Sprintf("[%s]", strings.Join(*sf, ","))
}

func (sf *SliceFlags) Set(value string) error {
	*sf = append(*sf, value)
	return nil
}
