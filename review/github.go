package review

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
)

// GitHub implements the Reviewer interface for GitHub PRs
type GitHub struct {
	*BaseReviewer
	client *github.Client
}

// NewGitHub creates a new GitHub reviewer client
func NewGitHub(opts ...Option) (Reviewer, error) {
	logger.Debug("Creating new GitHub reviewer client")

	baseReviewer, err := NewBaseReviewer(ProviderGitHub, opts...)
	if err != nil {
		return nil, err
	}

	if githubURL := os.Getenv("GITHUB_API_URL"); githubURL != "" {
		logger.Infof("Using GitHub Enterprise URL: %s", githubURL)
		baseReviewer.BaseURL = githubURL
	}

	gh := &GitHub{
		BaseReviewer: baseReviewer,
	}

	retryClient := common.NewRetryableClient(common.DefaultRetryConfig())
	standardClient := retryClient.StandardClient()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: gh.ApiToken})
	tc := oauth2.NewClient(context.Background(), ts)

	tc.Transport = &oauth2.Transport{
		Source: ts,
		Base:   standardClient.Transport,
	}

	if gh.BaseURL != "" {
		logger.Infof("Using GitHub Enterprise URL: %s", gh.BaseURL)
		apiURL, err := url.JoinPath(gh.BaseURL, "api/v3")
		if err != nil {
			errMsg := fmt.Sprintf("Failed to join API URL path: %v", err)
			logger.Errorf(errMsg)
			return nil, errors.New(errMsg)
		}
		uploadsURL, err := url.JoinPath(gh.BaseURL, "uploads")
		if err != nil {
			errMsg := fmt.Sprintf("Failed to join uploads URL path: %v", err)
			logger.Errorf(errMsg)
			return nil, errors.New(errMsg)
		}
		client, err := github.NewEnterpriseClient(apiURL, uploadsURL, tc)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create GitHub Enterprise client: %v", err)
			logger.Error(errMsg)
			return nil, errors.New(errMsg)
		}
		gh.client = client
	} else {
		gh.client = github.NewClient(tc)
	}

	logger.Debug("GitHub reviewer client created successfully")
	return gh, nil
}

// GetProvider returns the name of the review provider
func (gh *GitHub) GetProvider() string {
	return ProviderGitHub
}

func (gh *GitHub) SupportCollapsibleMarkdown() bool {
	return true
}

func (gh *GitHub) GetPullRequestDetails(repoOwner, repoName string, pr int) (common.PullRequest, error) {
	logger.Infof("Fetching pull request details for PR #%d in %s/%s", pr, repoOwner, repoName)
	ctx, cancel := gh.CreateTimeoutContext()
	defer cancel()

	if repoOwner == "" || repoName == "" || pr <= 0 {
		errMsg := "invalid repository owner, name or pull request number"
		logger.Error(errMsg)
		return common.PullRequest{}, errors.New(errMsg)
	}

	prDetails, _, err := gh.client.PullRequests.Get(ctx, repoOwner, repoName, pr)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get pull request details: %v", err)
		logger.Error(errMsg)
		return common.PullRequest{}, errors.New(errMsg)
	}

	title := prDetails.Title
	description := prDetails.Body
	owner := prDetails.User.GetName()
	createdAt := prDetails.CreatedAt.Format("2006-01-02 15:04:05")
	updatedAt := prDetails.UpdatedAt.Format("2006-01-02 15:04:05")
	merged := prDetails.Merged
	mergeable := prDetails.Mergeable

	labels := make([]common.Label, 0)
	for _, label := range prDetails.Labels {
		labels = append(labels, common.Label{
			Name: label.GetName(),
		})
	}

	commitsList, _, err := gh.client.PullRequests.ListCommits(ctx, repoOwner, repoName, pr, nil)
	if err != nil {
		errMsg := fmt.Sprintf("failed to list pull request commits: %v", err)
		logger.Error(errMsg)
		return common.PullRequest{}, errors.New(errMsg)
	}

	commits := make([]common.Commit, 0)
	for _, commit := range commitsList {
		commits = append(commits, common.Commit{
			CommitHash: commit.GetSHA(),
			Author:     commit.GetCommit().GetAuthor().GetName(),
			Message:    commit.GetCommit().GetMessage(),
		})
	}

	return common.PullRequest{
		Number:     pr,
		Title:      *title,
		Body:       *description,
		HeadBranch: prDetails.Head.GetRef(),
		BaseBranch: prDetails.Base.GetRef(),
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		Author:     owner,
		Mergeable:  mergeable != nil && *mergeable,
		Merged:     *merged,
		Labels:     labels,
		Commits:    commits,
	}, nil

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

func (gh *GitHub) getCommentBodyWithoutHeader(comments []*github.IssueComment, header string) (string, error) {
	// Check if summary already exists
	for _, c := range comments {
		if c.Body != nil && strings.HasPrefix(*c.Body, header) {
			body := strings.TrimPrefix(*c.Body, header)
			body = strings.TrimSpace(body)
			return body, nil
		}
	}
	return "", nil
}

func (gh *GitHub) ListComments(repoOwner, repoName string, pr int) ([]string, error) {
	ctx, cancel := gh.CreateTimeoutContext()
	defer cancel()

	comments, err := gh.getComments(ctx, repoOwner, repoName, pr)
	if err != nil {
		return nil, err
	}

	var commentBodies []string
	for _, c := range comments {
		if c.Body != nil {
			commentBodies = append(commentBodies, *c.Body)
		}

		// commentID := c.ID
		// replyTo := c.InReplyTo
		// c.GetReactions()
	}
	return commentBodies, nil
}

func (gh *GitHub) PostSummaryUnderReview(repoOwner, repoName string, pr int, header string) error {
	logger.Infof("Summary under update for PR #%d in %s/%s", pr, repoOwner, repoName)

	ctx, cancel := gh.CreateTimeoutContext()
	defer cancel()

	logger.Debug("Fetching existing comments to check for duplicates")
	comments, err := gh.getComments(ctx, repoOwner, repoName, pr)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to list existing comments: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	commentBody, err := gh.getCommentBodyWithoutHeader(comments, header)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to check existing comments: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	underReviewStr := common.Summary{}.InitiatedString(gh.GetProvider())
	err = gh.PostSummary(repoOwner, repoName, pr, header, underReviewStr+"\n\n"+commentBody)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to post summary under review: %v", err)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func (gh *GitHub) PostSummary(repoOwner, repoName string, pr int, header, body string) error {
	logger.Infof("Posting summary to PR #%d in %s/%s", pr, repoOwner, repoName)

	ctx, cancel := gh.CreateTimeoutContext()
	defer cancel()

	logger.Debug("Fetching existing comments to check for duplicates")
	comments, err := gh.getComments(ctx, repoOwner, repoName, pr)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to list existing comments: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	commentID, err := gh.getComment(comments, header)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to check existing comments: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	comment := &github.IssueComment{
		Body: &body,
	}

	if commentID > 0 {
		logger.Debugf("Found existing comment with ID: %d. Updating it", commentID)
		_, _, err = gh.client.Issues.EditComment(
			ctx,
			repoOwner,
			repoName,
			int64(commentID),
			comment,
		)

		if err != nil {
			errMsg := fmt.Sprintf("Failed to update existing summary comment: %v", err)
			logger.Error(errMsg)
			return errors.New(errMsg)
		}
		logger.Infof("Updated existing summary comment for PR #%d in %s/%s", pr, repoOwner, repoName)
	} else {
		_, _, err = gh.client.Issues.CreateComment(
			ctx,
			repoOwner,
			repoName,
			pr,
			comment,
		)

		if err != nil {
			errMsg := fmt.Sprintf("Failed to post summary comment: %v", err)
			logger.Error(errMsg)
			return errors.New(errMsg)
		}
		logger.Infof("Posted new summary comment for PR %d in %s/%s", pr, repoOwner, repoName)
	}

	return nil
}

func (gh *GitHub) PostLineFeedback(client *git.Client, repoOwner, repoName string, pr int, commitHash string, lineFeedback common.LineLevelFeedback) error {
	logger.Infof("Posting line feedback to PR #%d in %s/%s, commit: %s", pr, repoOwner, repoName, commitHash)

	ctx, cancel := gh.CreateTimeoutContext()
	defer cancel()

	// Check for existing comments
	logger.Debug("Fetching existing comments to check for duplicates")
	comments, err := gh.getComments(ctx, repoOwner, repoName, pr)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to list existing comments: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	reviewComments := make([]*github.DraftReviewComment, 0)

	logger.Debug("Getting existing review comments")
	addedComments, err := gh.GetReviewRequestComments(repoOwner, repoName, pr)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get existing review comments: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	logger.Infof("Processing %d line feedback items", len(lineFeedback.GetLineFeedback()))
	for _, ll := range lineFeedback.GetLineFeedback() {
		skip := false

		if ll.File == "" || ll.LineNumber <= 0 {
			logger.Warnf("Skipping invalid line feedback - file: %s, line: %d", ll.File, ll.LineNumber)
			continue
		}

		logger.Debugf("Getting blame for file: %s, line: %d", ll.File, ll.LineNumber)
		blame, err := client.GetBlameForFileLine(commitHash, ll.File, ll.LineNumber)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to get blame for line: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		for _, existingComment := range addedComments {
			if ll.File == existingComment.File &&
				ll.LineNumber >= existingComment.LineNumber && ll.LastLineNumber <= existingComment.LastLineNumber &&
				blame == existingComment.CommitHash {
				logger.Infof("Skipping existing comment for file: %s, line: %d", ll.File, ll.LineNumber)
				logger.Debugf("Existing comment:	line number: %d, last line number: %d, commit hash: %s", existingComment.LineNumber, existingComment.LastLineNumber, existingComment.CommitHash)
				logger.Debugf("Line feedback: 		line number: %d, last line number: %d, commit hash: %s", ll.LineNumber, ll.LastLineNumber, blame)
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		commentID, err := gh.getComment(comments, ll.Header(client, commitHash))
		if err != nil {
			errMsg := fmt.Sprintf("Failed to check existing comments: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		if commentID > 0 {
			continue
		}

		reviewBody := ll.String(gh.GetProvider(), client, commitHash)
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

	// Process nitpick comments
	var processErr error
	nitpickCommentsByFile, processErr := ProcessLineFeedbackItems(gh.GetProvider(), client, commitHash, addedComments, lineFeedback)
	if processErr != nil {
		errMsg := fmt.Sprintf("Failed to process line feedback items: %v", processErr)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	// Format nitpick comments for display
	nitpickComments := FormatNitpickComments(gh.GetProvider(), nitpickCommentsByFile)

	if len(reviewComments) > 0 || len(nitpickComments) > 0 {
		overallReviewStr := FormatOverallReview(len(reviewComments), nitpickComments)
		review := &github.PullRequestReviewRequest{
			CommitID: &commitHash,
			Body:     &overallReviewStr,
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
			errMsg := fmt.Sprintf("Failed to post line feedback: %v", err)
			logger.Error(errMsg)
			return errors.New(errMsg)
		}
		logger.Infof("Posted line feedback for PR %d in %s/%s", pr, repoOwner, repoName)
	}

	return nil
}

func (gh *GitHub) GetReviewRequestComments(repoOwner, repoName string, pr int) ([]common.LineLevel, error) {
	ctx, cancel := gh.CreateTimeoutContext()
	defer cancel()

	lineReviews := make([]common.LineLevel, 0)

	reviews, _, err := gh.client.PullRequests.ListReviews(ctx, repoOwner, repoName, pr, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to list reviews: %v", err)
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	for _, review := range reviews {
		if review.ID == nil {
			logger.Warn("Skipping review with nil ID")
			continue
		}

		comments, _, err := gh.client.PullRequests.ListReviewComments(ctx, repoOwner, repoName, pr, *review.ID, &github.ListOptions{})
		if err != nil {
			logger.Errorf("Failed to list review comments: %v", err)
			continue
		}

		for _, comment := range comments {
			// Skip replies to other comments
			if comment.InReplyTo != nil {
				logger.Debugf("Skipping reply to another comment: %d", *comment.InReplyTo)
				continue
			}
			if comment.PullRequestReviewID != nil && review.ID != nil && *comment.PullRequestReviewID == *review.ID {
				lines := strings.Split(*comment.Body, "\n")
				if len(lines) < 2 {
					logger.Debugf("Skipping comment with insufficient lines: %d", len(lines))
					continue
				}

				parts := strings.Split(lines[0], ":")
				if len(parts) < 4 {
					logger.Debugf("Skipping comment with insufficient parts: %d", len(parts))
					continue
				}

				file := strings.TrimSpace(parts[1])
				line := strings.TrimSpace(parts[2])

				var firstLine, lastLine int
				if strings.Contains(line, "-") {
					// Handle multi-line comments
					lineParts := strings.Split(line, "-")
					if len(lineParts) != 2 {
						logger.Debug("Skipping comment with invalid line range")
						continue
					}
					firstLine, _ = strconv.Atoi(strings.TrimSpace(lineParts[0]))
					lastLine, _ = strconv.Atoi(strings.TrimSpace(lineParts[1]))
				} else {
					firstLine, _ = strconv.Atoi(strings.TrimSpace(line))
					lastLine = firstLine
				}

				blame := strings.TrimSpace(parts[3])
				blame = strings.TrimSpace(strings.Split(blame, " ")[0])

				lineReviews = append(lineReviews, common.LineLevel{
					File:           file,
					LineNumber:     firstLine,
					LastLineNumber: lastLine,
					CommitHash:     blame,
					Body:           strings.Join(lines[1:], "\n"),
				})
			}
		}
	}

	return lineReviews, nil
}
