package common

import (
	"strconv"
	"strings"
)

type Label struct {
	Name string `yaml:"name"`
}

type Commit struct {
	CommitHash string `yaml:"commit_hash"`
	Author     string `yaml:"author"`
	Message    string `yaml:"message"`
	CreatedAt  string `yaml:"created_at"`
}

type PullRequest struct {
	Number     int      `yaml:"number"`
	Title      string   `yaml:"title"`
	Body       string   `yaml:"body"`
	HeadBranch string   `yaml:"head_branch"`
	BaseBranch string   `yaml:"base_branch"`
	CreatedAt  string   `yaml:"created_at"`
	UpdatedAt  string   `yaml:"updated_at"`
	Author     string   `yaml:"author"`
	Mergeable  bool     `yaml:"mergeable"`
	Merged     bool     `yaml:"merged"`
	Labels     []Label  `yaml:"labels"`
	Commits    []Commit `yaml:"commits"`
}

func (c Commit) String() string {
	return `Commit: ` + c.CommitHash + `
Author: ` + c.Author + `
Message: ` + c.Message + `
Created at: ` + c.CreatedAt + ``
}

func (pr PullRequest) String() string {
	labels := make([]string, len(pr.Labels))
	for i, label := range pr.Labels {
		labels[i] = label.Name
	}

	commits := make([]string, len(pr.Commits))
	for i, commit := range pr.Commits {
		commits[i] = commit.String()
	}

	return `Pull Request #` + strconv.Itoa(pr.Number) + `: ` + pr.Title + `
Author: ` + pr.Author + `
Created at: ` + pr.CreatedAt + `
Head Branch: ` + pr.HeadBranch + `
Base Branch: ` + pr.BaseBranch + `
Mergeable: ` + strconv.FormatBool(pr.Mergeable) + `
Merged: ` + strconv.FormatBool(pr.Merged) + `
Labels: ` + strings.Join(labels, ", ") + `
Commits:
` + strings.Join(commits, "\n\n") + `
Body: 
` + pr.Body
}
