package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// GitHubAdapter provides thin wrappers around the gh CLI for Issue/PR operations.
type GitHubAdapter struct {
	repo                    string
	ghCLI                   string
	dryRun                  bool
	ForceRequestReviewError error // test hook
}

// NewGitHubAdapter constructs a GitHub adapter bound to one repository.
func NewGitHubAdapter(repo, ghCLI string, dryRun bool) *GitHubAdapter {
	if ghCLI == "" {
		ghCLI = "gh"
	}
	return &GitHubAdapter{repo: repo, ghCLI: ghCLI, dryRun: dryRun}
}

// ReadyIssue represents a minimal Issue view needed by the dispatcher.
type ReadyIssue struct {
	Number int64
	Title  string
	Body   string
	Labels []string
}

// ListReadyIssues returns issues labeled with route:<role> and state:ready.
func (a *GitHubAdapter) ListReadyIssues(ctx context.Context, role string) ([]ReadyIssue, error) {
	if a.dryRun {
		return []ReadyIssue{{Number: 1, Title: "dry-run issue", Body: "body", Labels: []string{fmt.Sprintf("route:%s", role), "state:ready"}}}, nil
	}
	routeLabel := fmt.Sprintf("route:%s", role)
	stateLabel := "state:ready"
	args := []string{"issue", "list", "--repo", a.repo, "--label", routeLabel, "--label", stateLabel, "--state", "open", "--json", "number,title,body,labels", "--limit", "100"}
	out, err := a.run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("list ready issues: %w", err)
	}
	var raw []struct {
		Number int64
		Title  string
		Body   string
		Labels []struct{ Name string }
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("decode issues: %w", err)
	}
	outIssues := make([]ReadyIssue, 0, len(raw))
	for _, i := range raw {
		labels := make([]string, 0, len(i.Labels))
		for _, l := range i.Labels {
			labels = append(labels, l.Name)
		}
		outIssues = append(outIssues, ReadyIssue{Number: i.Number, Title: i.Title, Body: i.Body, Labels: labels})
	}
	return outIssues, nil
}

// TransitionToWorking replaces ready state label with working state label.
func (a *GitHubAdapter) TransitionToWorking(ctx context.Context, issueNumber int64, role string) error {
	routeLabel := fmt.Sprintf("route:%s", role)
	ready := "state:ready"
	working := "state:working"
	if a.dryRun {
		return nil
	}
	if _, err := a.run(ctx, "issue", "edit", fmt.Sprintf("%d", issueNumber), "--repo", a.repo, "--remove-label", ready, "--add-label", working); err != nil {
		return fmt.Errorf("transition to working: %w", err)
	}
	_ = routeLabel
	return nil
}

// TransitionToReview replaces working state label with review state label.
func (a *GitHubAdapter) TransitionToReview(ctx context.Context, issueNumber int64) error {
	if a.dryRun {
		return nil
	}
	if _, err := a.run(ctx, "issue", "edit", fmt.Sprintf("%d", issueNumber), "--repo", a.repo, "--remove-label", "state:working", "--add-label", "state:review"); err != nil {
		return fmt.Errorf("transition to review: %w", err)
	}
	return nil
}

// AddComment posts a comment to an issue or PR.
func (a *GitHubAdapter) AddComment(ctx context.Context, number int64, body string) error {
	if a.dryRun {
		return nil
	}
	if _, err := a.run(ctx, "issue", "comment", fmt.Sprintf("%d", number), "--repo", a.repo, "--body", body); err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	return nil
}

// CreatePR creates a pull request and returns its number.
func (a *GitHubAdapter) CreatePR(ctx context.Context, title, body, head, base string) (int64, error) {
	if a.dryRun {
		return -1, nil
	}
	args := []string{"pr", "create", "--repo", a.repo, "--title", title, "--body", body, "--head", head, "--base", base}
	out, err := a.run(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("create pr: %w", err)
	}
	// gh pr create outputs the PR URL on the last line when not in JSON mode.
	text := strings.TrimSpace(string(out))
	if strings.HasPrefix(text, "https://") {
		parts := strings.Split(text, "/")
		num := parts[len(parts)-1]
		var n int64
		if _, err := fmt.Sscan(num, &n); err != nil {
			return 0, fmt.Errorf("parse pr number from %q: %w", text, err)
		}
		return n, nil
	}
	return 0, fmt.Errorf("unexpected pr create output: %s", text)
}

// RequestReview requests a review from a role label on a PR.
func (a *GitHubAdapter) RequestReview(ctx context.Context, prNumber int64, reviewRole string) error {
	if a.ForceRequestReviewError != nil {
		return a.ForceRequestReviewError
	}
	if a.dryRun {
		return nil
	}
	// Use the review role label as a pseudo-request; real GitHub review requests
	// require a user/login, which the dispatcher does not own.
	body := fmt.Sprintf("[dispatcher] review requested for role: %s", reviewRole)
	if err := a.AddComment(ctx, prNumber, body); err != nil {
		return err
	}
	label := fmt.Sprintf("review:%s", reviewRole)
	if _, err := a.run(ctx, "pr", "edit", fmt.Sprintf("%d", prNumber), "--repo", a.repo, "--add-label", label); err != nil {
		return fmt.Errorf("add review label: %w", err)
	}
	return nil
}

// GetHeadSHA returns the current HEAD SHA of the repository.
func (a *GitHubAdapter) GetHeadSHA(ctx context.Context) (string, error) {
	out, err := a.run(ctx, "repo", "view", "--repo", a.repo, "--json", "object")
	if err != nil {
		return "", fmt.Errorf("repo view: %w", err)
	}
	var view struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal(out, &view); err != nil {
		return "", fmt.Errorf("decode repo view: %w", err)
	}
	return view.Object.SHA, nil
}

func (a *GitHubAdapter) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, a.ghCLI, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "HTTP 404") || strings.Contains(string(out), "Could not resolve to a node") {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// IsDryRun reports whether the adapter is configured for dry-run mode.
func (a *GitHubAdapter) IsDryRun() bool {
	return a.dryRun
}

// ErrRepoNotAllowed is returned when an operation targets a repo outside allowedRepos.
var ErrRepoNotAllowed = errors.New("repository not allowed by config")
