package review

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
)

// GitHub implements the Reviewer interface for GitHub PRs
type GitHub struct {
	client   *github.Client
	apiToken string
	timeout  int
}

// NewGitHub creates a new GitHub reviewer client
func NewGitHub(opts ...Option) (Reviewer, error) {
	gh := &GitHub{
		timeout: 60, // Default timeout
	}

	// Apply options
	for _, opt := range opts {
		switch opt.Type {
		case APITokenOption:
			if token, ok := opt.Value.(string); ok {
				gh.apiToken = token
			}
		case TimeoutOption:
			if timeout, ok := opt.Value.(int); ok {
				gh.timeout = timeout
			}
		}
	}

	// Validate required options
	if gh.apiToken == "" {
		return nil, fmt.Errorf("API token is required for GitHub")
	}

	// Create GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: gh.apiToken})
	tc := oauth2.NewClient(context.Background(), ts)
	gh.client = github.NewClient(tc)

	return gh, nil
}

// PostReview submits review comments to GitHub PR
func (gh *GitHub) PostReview(req ReviewRequest) ReviewResponse {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(gh.timeout)*time.Second)
	defer cancel()

	// First, add the summary as a PR comment
	comment := &github.IssueComment{
		Body: &req.Summary,
	}

	_, _, err := gh.client.Issues.CreateComment(
		ctx,
		req.RepoOwner(),
		req.RepoName(),
		req.PRNumber,
		comment,
	)

	if err != nil {
		return ReviewResponse{
			Success: false,
			Error:   fmt.Errorf("failed to post summary comment: %w", err),
		}
	}

	// Then add review comments if any
	if len(req.Comments) > 0 {
		// Create review comments
		comments := make([]*github.DraftReviewComment, 0, len(req.Comments))

		for _, c := range req.Comments {
			comments = append(comments, &github.DraftReviewComment{
				Path:     &c.FilePath,
				Body:     &c.Body,
				Position: github.Int(c.Line),
			})
		}

		review := &github.PullRequestReviewRequest{
			CommitID: nil, // Uses the latest commit
			Body:     &req.Summary,
			Event:    github.String("COMMENT"),
			Comments: comments,
		}

		_, _, err = gh.client.PullRequests.CreateReview(
			ctx,
			req.RepoOwner(),
			req.RepoName(),
			req.PRNumber,
			review,
		)

		if err != nil {
			return ReviewResponse{
				Success: false,
				Error:   fmt.Errorf("failed to post review comments: %w", err),
			}
		}
	}

	// Construct the PR URL
	prURL := fmt.Sprintf("https://github.com/%s/pull/%d", req.Repository, req.PRNumber)

	return ReviewResponse{
		Success: true,
		URL:     prURL,
		Error:   nil,
	}
}
