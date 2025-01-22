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

	// Find release trains
	query := fmt.Sprintf("attr(deployment_branch, \".+\", attr(release_branch_prefix, \"%s\", kind(gitops, %s)))",
		cfg.ReleaseBranch, cfg.Targets)

	result := executeBazelQuery(query)

	trains := make(map[string][]string)
	for _, t := range result.Results {
		for _, attr := range t.Target.GetRule().GetAttribute() {
			if attr.GetName() == "deployment_branch" {
				trains[attr.GetStringValue()] = append(trains[attr.GetStringValue()], t.Target.Rule.GetName())
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

		if !workdir.IsClean() {
			log.Println("executing get modified files")
			log.Println(workdir.GetModifiedFiles())
			log.Println("finishing get modified files")
		}

		files, err := workdir.GetModifiedFiles()

		if err != nil {
			log.Fatalf("failed to get modified files: %v", err)
		}

		modifiedFiles = append(modifiedFiles, files...)

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

	processImages(updatedTargets, cfg)

	if !cfg.DryRun {
		commitMsg := fmt.Sprintf("GitOps for release branch %s from %s commit %s\n",
			cfg.ReleaseBranch, cfg.BranchName, cfg.GitCommit)
		github_app.CreateCommit(cfg.PRTargetBranch, cfg.BranchName, gitopsDir, modifiedFiles, commitMsg)
	}

	// createPullRequests(updatedBranches, cfg)
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
