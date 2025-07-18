package llm

type EnabledTools struct {
	ListDirectory         bool
	GetGitDiff            bool
	ReadFile              bool
	SearchCodebase        bool
	GetGitBlame           bool
	GetPullRequestDetails bool
	GetBuildLog           bool
	PostSummary           bool
	PostLineFeedback      bool
	PostBuildSummary      bool
}
