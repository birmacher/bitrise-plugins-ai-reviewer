package review

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
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

func (gh *GitHub) getComments(ctx context.Context, repoOwner, repoName string, pr int) ([]*github.IssueComment, error) {
	comments, _, err := gh.client.Issues.ListComments(
		ctx,
		repoOwner,
		repoName,
		pr,
		nil,
	)
	return comments, err
}

func (gh *GitHub) getComment(comments []*github.IssueComment, header string) (int64, error) {
	// Check if summary already exists
	for _, c := range comments {
		if c.Body != nil && strings.HasPrefix(*c.Body, header) {
			return *c.ID, nil
		}
	}
	return 0, nil
}

func (gh *GitHub) PostSummary(repoOwner, repoName string, pr int, summary common.Summary) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(gh.timeout)*time.Second)
	defer cancel()

	comments, err := gh.getComments(ctx, repoOwner, repoName, pr)
	if err != nil {
		return fmt.Errorf("failed to list existing comments: %w", err)
	}

	commentID, err := gh.getComment(comments, summary.Header())
	if err != nil {
		return fmt.Errorf("failed to check existing comments: %w", err)
	}

	commentBody := summary.String()
	comment := &github.IssueComment{
		Body: &commentBody,
	}

	if commentID > 0 {
		_, _, err = gh.client.Issues.EditComment(
			ctx,
			repoOwner,
			repoName,
			int64(commentID),
			comment,
		)

		if err != nil {
			return fmt.Errorf("failed to update existing summary comment: %w", err)
		}
	} else {
		_, _, err = gh.client.Issues.CreateComment(
			ctx,
			repoOwner,
			repoName,
			pr,
			comment,
		)

		if err != nil {
			return fmt.Errorf("failed to post summary comment: %w", err)
		}
	}

	return nil
}

func (gh *GitHub) PostLineFeedback(client *git.Client, repoOwner, repoName string, pr int, commitHash string, lineFeedback common.LineLevelFeedback) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(gh.timeout)*time.Second)
	defer cancel()

	// Check for existing comments
	comments, err := gh.getComments(ctx, repoOwner, repoName, pr)
	if err != nil {
		return fmt.Errorf("failed to list existing comments: %w", err)
	}

	reviewComments := make([]*github.DraftReviewComment, 0)

	for _, ll := range lineFeedback.Lines {
		commentID, err := gh.getComment(comments, ll.Header(client, commitHash))

		if err != nil {
			return fmt.Errorf("failed to check existing comments: %w", err)
		}

		if commentID > 0 {
			continue
		}

		if ll.File == "" || ll.LineNumber <= 0 {
			continue
		}

		reviewBody := ll.String(client, commitHash)
		reviewComment := &github.DraftReviewComment{
			Path: &ll.File,
			Line: &ll.LineNumber,
			Body: &reviewBody,
			Side: github.String("RIGHT"), // Always set the side to RIGHT for new file content
		}
		if ll.LastLineNumber > 0 && ll.LastLineNumber > ll.LineNumber {
			reviewComment.StartLine = &ll.LineNumber
			reviewComment.Line = &ll.LastLineNumber
		}

		reviewComments = append(reviewComments, reviewComment)
	}

	if len(reviewComments) > 0 {
		overallReview := "This is an AI-generated review. Please review it carefully."

		review := &github.PullRequestReviewRequest{
			CommitID: &commitHash,
			Body:     &overallReview,
			Event:    github.String("COMMENT"),
			Comments: reviewComments,
		}

		_, _, err := gh.client.PullRequests.CreateReview(
			ctx,
			repoOwner,
			repoName,
			pr,
			review,
		)

		if err != nil {
			return fmt.Errorf("failed to post line feedback: %w", err)
		}
	}

	return nil
}

func (gh *GitHub) GetReviewRequestComments(repoOwner, repoName string, pr int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(gh.timeout)*time.Second)
	defer cancel()

	reviews, _, err := gh.client.PullRequests.ListReviews(ctx, repoOwner, repoName, pr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list reviews: %w", err)
	}

	var sb strings.Builder

	for _, review := range reviews {
		reviewID := review.ID
		if reviewID == nil {
			continue
		}

		comments, _, err := gh.client.PullRequests.ListReviewComments(ctx, repoOwner, repoName, pr, *reviewID, &github.ListOptions{})
		if err != nil {
			fmt.Println("Failed to list review comments:", err)
			continue
		}

		for _, comment := range comments {
			// Skip replies to other comments
			if comment.InReplyTo != nil {
				continue
			}
			if comment.PullRequestReviewID != nil && review.ID != nil && *comment.PullRequestReviewID == *review.ID {
				lines := strings.Split(*comment.Body, "\n")
				if len(lines) < 2 {
					continue
				}

				parts := strings.Split(lines[0], ":")
				if len(parts) < 3 {
					continue
				}

				sb.WriteString(fmt.Sprintf("===== Line Level Review: file: %s lines: %s =====\n", parts[1], parts[2]))

				if comment.Body != nil {
					sb.WriteString(strings.Join(lines[1:], "\n"))
					sb.WriteString("\n")
				}
				sb.WriteString("===== END =====\n\n")
			}
		}
	}

	return sb.String(), nil
}
