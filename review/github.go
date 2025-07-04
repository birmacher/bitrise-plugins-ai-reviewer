package review

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
)

// GitHub implements the Reviewer interface for GitHub PRs
type GitHub struct {
	client   *github.Client
	apiToken string
	timeout  int
	baseURL  string
}

// NewGitHub creates a new GitHub reviewer client
func NewGitHub(opts ...Option) (Reviewer, error) {
	logger.Debug("Creating new GitHub reviewer client")

	gh := &GitHub{
		timeout: 60, // Default timeout
	}

	// Apply options
	for _, opt := range opts {
		switch opt.Type {
		case APITokenOption:
			if token, ok := opt.Value.(string); ok {
				gh.apiToken = token
				logger.Debug("GitHub API token configured")
			}
		case TimeoutOption:
			if timeout, ok := opt.Value.(int); ok {
				gh.timeout = timeout
				logger.Debugf("GitHub API timeout set to %d seconds", timeout)
			}
		case BaseURLOption:
			if baseURL, ok := opt.Value.(string); ok {
				gh.baseURL = baseURL
				logger.Debugf("GitHub Enterprise base URL configured: %s", baseURL)
			}
		}
	}

	// Validate required options
	if gh.apiToken == "" {
		errMsg := "GitHub API token is required"
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	retryClient := common.NewRetryableClient(common.DefaultRetryConfig())
	standardClient := retryClient.StandardClient()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: gh.apiToken})
	tc := oauth2.NewClient(context.Background(), ts)

	tc.Transport = &oauth2.Transport{
		Source: ts,
		Base:   standardClient.Transport,
	}

	if gh.baseURL != "" {
		logger.Infof("Using GitHub Enterprise URL: %s", gh.baseURL)
		apiURL, err := url.JoinPath(gh.baseURL, "api/v3")
		if err != nil {
			errMsg := fmt.Sprintf("Failed to join API URL path: %v", err)
			logger.Errorf(errMsg)
			return nil, errors.New(errMsg)
		}
		uploadsURL, err := url.JoinPath(gh.baseURL, "uploads")
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

func (gh *GitHub) PostSummary(repoOwner, repoName string, pr int, header, body string) error {
	logger.Infof("Posting summary to PR #%d in %s/%s", pr, repoOwner, repoName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(gh.timeout)*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(gh.timeout)*time.Second)
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
	nitpickComments := make([]string, 0)
	nitpickCommentsByFile := make(map[string][]common.LineLevel)

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

	// Nitpick comment
	for _, ll := range lineFeedback.GetNitpickFeedback() {
		if ll.File == "" || ll.LineNumber <= 0 {
			continue
		}
		if nitpickCommentsByFile[ll.File] == nil {
			nitpickCommentsByFile[ll.File] = []common.LineLevel{}
		}
		nitpickCommentsByFile[ll.File] = append(nitpickCommentsByFile[ll.File], ll)
	}

	nitpickComment := strings.Builder{}
	for filepath, comments := range nitpickCommentsByFile {
		nitpickComment.WriteString("<details>\n")
		nitpickComment.WriteString("<summary>" + filepath + " (" + strconv.Itoa(len(comments)) + ")</summary>\n\n")
		for _, c := range comments {
			line := fmt.Sprintf("%d", c.LineNumber)
			if c.IsMultiline() {
				line = line + "-" + fmt.Sprintf("%d", c.LastLineNumber)
			}
			nitpickComment.WriteString("<!-- bitrise-plugin-ai-reviewer: " + filepath + ":" + line + " -->\n")
			nitpickComment.WriteString("`" + line + "`: **" + c.Title + "**\n\n")
			nitpickComment.WriteString(c.Body + "\n\n")
		}
		nitpickComment.WriteString("</details>\n\n")

		nitpickComments = append(nitpickComments, nitpickComment.String())
	}

	if len(reviewComments) > 0 {
		overallReview := strings.Builder{}
		overallReview.WriteString("_This is an AI-generated review. Please review it carefully._\n\n")
		overallReview.WriteString(fmt.Sprintf("**Actionable comments posted: %d**\n\n", len(reviewComments)))
		if len(nitpickComments) > 0 {
			overallReview.WriteString("<details>\n")
			overallReview.WriteString("<summary>🧹 Nitpick comments</summary>\n")
			overallReview.WriteString(strings.Join(nitpickComments, "\n\n---\n\n"))
			overallReview.WriteString("</details>\n\n")
		}

		overallReviewStr := overallReview.String()
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(gh.timeout)*time.Second)
	defer cancel()

	lineReviews := make([]common.LineLevel, 0)

	reviews, _, err := gh.client.PullRequests.ListReviews(ctx, repoOwner, repoName, pr, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to list reviews: %v", err)
		fmt.Println(errMsg)
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
