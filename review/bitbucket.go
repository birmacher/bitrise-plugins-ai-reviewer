package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"golang.org/x/oauth2"
)

// Bitbucket implements the Reviewer interface for Bitbucket PRs
type Bitbucket struct {
	*BaseReviewer
	client *http.Client
}

// PRComment represents a comment on a pull request
type PRComment struct {
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	Inline struct {
		Path string `json:"path"`
		From int    `json:"from,omitempty"`
		To   int    `json:"to,omitempty"`
	} `json:"inline,omitempty"`
}

// CommentResponse represents a response from the Bitbucket API for comment operations
type CommentResponse struct {
	ID      int `json:"id"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	CreatedOn time.Time `json:"created_on"`
}

// NewBitbucket creates a new Bitbucket reviewer client
func NewBitbucket(opts ...Option) (Reviewer, error) {
	logger.Debug("Creating new Bitbucket reviewer client")

	baseReviewer, err := NewBaseReviewer(ProviderBitbucket, opts...)
	if err != nil {
		return nil, err
	}

	baseReviewer.BaseURL = "https://api.bitbucket.org/2.0"

	bb := &Bitbucket{
		BaseReviewer: baseReviewer,
	}

	retryClient := common.NewRetryableClient(common.DefaultRetryConfig())
	standardClient := retryClient.StandardClient()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: bb.ApiToken})
	tc := oauth2.NewClient(context.Background(), ts)

	tc.Transport = &oauth2.Transport{
		Source: ts,
		Base:   standardClient.Transport,
	}

	bb.client = tc

	logger.Debug("Bitbucket reviewer client created successfully")
	return bb, nil
}

// GetProvider returns the name of the review provider
func (bb *Bitbucket) GetProvider() string {
	return ProviderBitbucket
}

func (bb *Bitbucket) SupportCollapsibleMarkdown() bool {
	return false
}

// getComments retrieves all comments for a pull request
func (bb *Bitbucket) getComments(ctx context.Context, repoOwner, repoName string, pr int) ([]CommentResponse, error) {
	commentsURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments",
		bb.BaseURL, repoOwner, repoName, pr)

	req, err := http.NewRequestWithContext(ctx, "GET", commentsURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := bb.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get comments: HTTP %d", resp.StatusCode)
	}

	// Parse the response
	var response struct {
		Values []CommentResponse `json:"values"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Values, nil
}

// getComment checks if a comment with the specified header already exists
func (bb *Bitbucket) getComment(comments []CommentResponse, header string) (int, string, error) {
	for _, c := range comments {
		if strings.HasPrefix(c.Content.Raw, header) {
			return c.ID, c.Content.Raw, nil
		}
	}
	return 0, "", nil
}

// PostSummary adds or updates a summary comment on a Bitbucket pull request
func (bb *Bitbucket) PostSummary(repoOwner, repoName string, pr int, header, body string) error {
	logger.Infof("Posting summary to PR #%d in %s/%s", pr, repoOwner, repoName)

	ctx, cancel := bb.CreateTimeoutContext()
	defer cancel()

	logger.Debug("Fetching existing comments to check for duplicates")
	comments, err := bb.getComments(ctx, repoOwner, repoName, pr)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to list existing comments: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	commentID, commentBody, err := bb.getComment(comments, header)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to check existing comments: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	// For regular summary comments (without inline feedback), we can use a simpler structure
	// that only includes the content field for both new and updated comments
	commentData := struct {
		Content struct {
			Raw string `json:"raw"`
		} `json:"content"`
	}{
		Content: struct {
			Raw string `json:"raw"`
		}{
			Raw: body,
		},
	}

	jsonData, err := json.Marshal(commentData)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to marshal comment data: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	var req *http.Request
	var apiURL string

	if commentID > 0 {
		// Update existing comment
		apiURL = fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments/%d",
			bb.BaseURL, repoOwner, repoName, pr, commentID)

		commentData.Content.Raw = commentData.Content.Raw + "\n\n" + commentBody

		req, err = http.NewRequestWithContext(ctx, "PUT", apiURL, strings.NewReader(string(jsonData)))
		logger.Debugf("Updating existing comment with ID: %d", commentID)
	} else {
		// Create new comment
		apiURL = fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments",
			bb.BaseURL, repoOwner, repoName, pr)
		req, err = http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonData)))
		logger.Debug("Creating new comment")
	}

	if err != nil {
		errMsg := fmt.Sprintf("Failed to create request: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := bb.client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send request: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		errMsg := fmt.Sprintf("Failed to post comment: HTTP %d", resp.StatusCode)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	if commentID > 0 {
		logger.Infof("Updated existing summary comment for PR #%d in %s/%s", pr, repoOwner, repoName)
	} else {
		logger.Infof("Posted new summary comment for PR #%d in %s/%s", pr, repoOwner, repoName)
	}

	return nil
}

// PostLineFeedback adds line-specific review comments to a Bitbucket pull request
func (bb *Bitbucket) PostLineFeedback(client *git.Client, repoOwner, repoName string, pr int, commitHash string, lineFeedback common.LineLevelFeedback) error {
	logger.Infof("Posting line feedback to PR #%d in %s/%s, commit: %s", pr, repoOwner, repoName, commitHash)

	ctx, cancel := bb.CreateTimeoutContext()
	defer cancel()

	// Get existing comments to avoid duplicates
	logger.Debug("Fetching existing comments to check for duplicates")
	comments, err := bb.getComments(ctx, repoOwner, repoName, pr)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to list existing comments: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	// Get existing review comments
	logger.Debug("Getting existing review comments")
	existingComments, err := bb.GetReviewRequestComments(repoOwner, repoName, pr)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get existing review comments: %v", err)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	// Track feedback that will be posted
	lineComments := []PRComment{}
	nitpickComments := []string{}
	nitpickCommentsByFile := make(map[string][]common.LineLevel)

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

		// Check if we already have a comment for this location
		for _, existingComment := range existingComments {
			if ll.File == existingComment.File &&
				ll.LineNumber >= existingComment.LineNumber && ll.LastLineNumber <= existingComment.LastLineNumber &&
				blame == existingComment.CommitHash {
				logger.Infof("Skipping existing comment for file: %s, line: %d", ll.File, ll.LineNumber)
				logger.Debugf("Existing comment: line number: %d, last line number: %d, commit hash: %s", existingComment.LineNumber, existingComment.LastLineNumber, existingComment.CommitHash)
				logger.Debugf("Line feedback: line number: %d, last line number: %d, commit hash: %s", ll.LineNumber, ll.LastLineNumber, blame)
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		// Check if we've already added this comment in this PR
		commentID, _, err := bb.getComment(comments, ll.Header(client, commitHash))
		if err != nil {
			errMsg := fmt.Sprintf("Failed to check existing comments: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		if commentID > 0 {
			continue
		}

		// Prepare the inline comment
		reviewBody := ll.String(bb.GetProvider(), client, commitHash)
		comment := PRComment{
			Content: struct {
				Raw string `json:"raw"`
			}{
				Raw: reviewBody,
			},
			Inline: struct {
				Path string `json:"path"`
				From int    `json:"from,omitempty"`
				To   int    `json:"to,omitempty"`
			}{
				Path: ll.File,
				To:   ll.LineNumber,
			},
		}

		// Handle multiline comments
		if ll.LastLineNumber > 0 && ll.LastLineNumber > ll.LineNumber {
			comment.Inline.From = ll.LineNumber
			comment.Inline.To = ll.LastLineNumber
		}

		lineComments = append(lineComments, comment)
	}

	// Process nitpick comments
	var processErr error
	nitpickCommentsByFile, processErr = ProcessLineFeedbackItems(bb.GetProvider(), client, commitHash, existingComments, lineFeedback)
	if processErr != nil {
		errMsg := fmt.Sprintf("Failed to process line feedback items: %v", processErr)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	// Format nitpick comments for display
	nitpickComments = FormatNitpickComments(bb.GetProvider(), nitpickCommentsByFile)

	// Post all line comments
	if len(lineComments) > 0 {
		for _, comment := range lineComments {
			jsonData, err := json.Marshal(comment)
			if err != nil {
				logger.Errorf("Failed to marshal comment data: %v", err)
				continue
			}

			apiURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments",
				bb.BaseURL, repoOwner, repoName, pr)

			req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonData)))
			if err != nil {
				logger.Errorf("Failed to create request: %v", err)
				continue
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := bb.client.Do(req)
			if err != nil {
				logger.Errorf("Failed to send request: %v", err)
				continue
			}

			if resp.StatusCode >= 300 {
				logger.Errorf("Failed to post comment: HTTP %d", resp.StatusCode)
			}

			resp.Body.Close()
		}
	}

	// Post nitpick comments as a summary comment if they exist
	if len(nitpickComments) > 0 {
		overallReviewStr := FormatOverallReview(len(lineComments), nitpickComments)

		nitpickPRComment := PRComment{
			Content: struct {
				Raw string `json:"raw"`
			}{
				Raw: overallReviewStr,
			},
		}

		jsonData, err := json.Marshal(nitpickPRComment)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to marshal nitpick comment data: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		apiURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments",
			bb.BaseURL, repoOwner, repoName, pr)

		req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonData)))
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create request for nitpick comments: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := bb.client.Do(req)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to send request for nitpick comments: %v", err)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			errMsg := fmt.Sprintf("Failed to post nitpick comments: HTTP %d", resp.StatusCode)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}
	}

	logger.Infof("Posted line feedback for PR %d in %s/%s", pr, repoOwner, repoName)
	return nil
}

// GetReviewRequestComments retrieves existing review comments for a PR
func (bb *Bitbucket) GetReviewRequestComments(repoOwner, repoName string, pr int) ([]common.LineLevel, error) {
	ctx, cancel := bb.CreateTimeoutContext()
	defer cancel()

	lineReviews := []common.LineLevel{}

	apiURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments?fields=values.content,values.inline,values.id",
		bb.BaseURL, repoOwner, repoName, pr)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create request for comments: %v", err)
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	resp, err := bb.client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get comments: %v", err)
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Failed to get comments: HTTP %d", resp.StatusCode)
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	type Comment struct {
		ID      int `json:"id"`
		Content struct {
			Raw string `json:"raw"`
		} `json:"content"`
		Inline struct {
			Path string `json:"path,omitempty"`
			From int    `json:"from,omitempty"`
			To   int    `json:"to,omitempty"`
		} `json:"inline,omitempty"`
	}

	var commentsResponse struct {
		Values []Comment `json:"values"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commentsResponse); err != nil {
		errMsg := fmt.Sprintf("Failed to decode comment response: %v", err)
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	for _, comment := range commentsResponse.Values {
		// Skip comments without inline information
		if comment.Inline.Path == "" {
			continue
		}

		lines := strings.Split(comment.Content.Raw, "\n")
		if len(lines) < 2 {
			continue
		}

		// Parse comment header to extract metadata
		headerLine := lines[0]
		if !strings.Contains(headerLine, "bitrise-plugin-ai-reviewer") {
			continue
		}

		parts := strings.Split(headerLine, ":")
		if len(parts) < 4 {
			continue
		}

		file := strings.TrimSpace(parts[1])
		line := strings.TrimSpace(parts[2])

		var firstLine, lastLine int
		if strings.Contains(line, "-") {
			// Handle multi-line comments
			lineParts := strings.Split(line, "-")
			if len(lineParts) != 2 {
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

	return lineReviews, nil
}
