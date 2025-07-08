# Bitrise AI Reviewer Plugin

A Bitrise CLI plugin for reviewing and summarizing code changes using AI.

## Overview

This plugin helps to automate code reviews by using AI to analyze pull requests and provide feedback, suggestions, and potential issue detection. It uses the OpenAI and Anthropic API to generate insights about your code changes, with structured responses focusing on code quality, potential bugs, and performance issues.

## Features

- **PR Review**: Analyze GitHub and Bitbucket pull requests for potential issues
- **Code Summarization**: Generate concise summaries of code changes
- **Line-by-Line Feedback**: Get specific feedback on individual code lines
- **Integration with GitHub**: Automatically fetch PR details and provide feedback

## Installation

### Install plugin

```bash
bitrise plugin install --source https://github.com/bitrise-io/bitrise-plugins-ai-reviewer
```

### Install on Bitrise

#### bitrise.yml

This will be executed for all PR changes

**Make sure to set this report "AI Review" as non-blocking on GitHub for merges**

```yml
#workflows:
  ai_pr_summary:
    triggers:
      # run for pull requests; changed_files filter exposes the list of changed files
      pull_request:
      - target_branch: '*'
        source_branch: '*'
        changed_files: '*'
    # Set status_report_name to report the status of this workflow separately.
    status_report_name: 'AI Review'
    # Simple Medium Linux machine is enough
    meta:
      bitrise.io:
        machine_type_id: g2.linux.medium
        stack: linux-docker-android-22.04
    envs:
    - GITHUB_TOKEN: $ADD_GITHUB_ACCESS_TOKEN
    - LLM_API_KEY: $ADD_OPENAI_ACCESS_TOKEN
    steps:
    - activate-ssh-key@4:
        run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
    - git-clone@8.4.0: {}
    - script@1.2.1:
        title: Generate AI Review for PR
        inputs:
        - content: |-            
            #!/bin/bash
            set -e

            # Parse repository name from repo URL (works for SSH & HTTPS)
            REPO_URL="${GIT_REPOSITORY_URL}"
            REPO=$(echo "$REPO_URL" | sed -E 's#(git@|https://)([^/:]+)[/:]([^/]+)/([^.]+)(\.git)?#\3/\4#')

            # 1. Unshallow the repo if it's a shallow clone (safe to run even if already full)
            git fetch --unshallow || true

            # 2. Fetch all branch refs (this ensures both the PR and the target/destination branch are present)
            git fetch origin

            # 3. Fetch both relevant branches explicitly for safety (redundant but safe)
            git fetch origin "$BITRISEIO_GIT_BRANCH_DEST"
            git fetch origin "$BITRISE_GIT_BRANCH"

            # 4. Create/reset local branches to match the remote
            git checkout -B "$BITRISEIO_GIT_BRANCH_DEST" "origin/$BITRISEIO_GIT_BRANCH_DEST"
            git checkout -B "$BITRISE_GIT_BRANCH" "origin/$BITRISE_GIT_BRANCH"

            # (Optionally: check out the PR branch if that is the branch you want to analyze)
            git checkout "$BITRISE_GIT_BRANCH"

            # 5. Install AI reviewer plugin (customize source as needed)
            bitrise plugin install --source https://github.com/bitrise-io/bitrise-plugins-ai-reviewer.git

            # 6. Run your AI reviewer (customize flags as needed)
            bitrise :ai-reviewer summarize \
              -m=gpt-4.1 \
              -c="${GIT_CLONE_COMMIT_HASH}" \
              -b="${BITRISEIO_GIT_BRANCH_DEST}" \
              -r=github \
              --pr="${BITRISE_PULL_REQUEST}" \
              --repo="${REPO}"
              --log-level=debug

            echo "Done! PR reviewed."
```

#### Configure the plugin

You can add a `review.bitrise.yml` file to the root directory to configure the plugin

```yml
language: "en-US"               # language to use
tone_instructions: ""           # any additional instruction for the LLM on how to respond
reviews:
  profile: "chill"              # can be chill or assertive
  summary: true                 # should it generate summary
  walkthrough: true             # should it generate walkthrough
  collapse_walkthrough: true    # should the summary and walkthrough collapsed
  haiku: true                   # should it generate a haiku
  path_filters: ""              # todo
  path_instructions: ""         # todo
```

## Configuration

Set up your environment with the necessary API tokens:

```bash
export GITHUB_TOKEN=your_github_personal_access_token
export LLM_API_KEY=your_openai_api_key
```

For GitHub Enterprise, you can configure the API URL:

```bash
export GITHUB_API_URL=https://github.yourdomain.com
```

## Usage

### Review a Pull Request

```bash
bitrise ai-reviewer review --pr <PR_NUMBER> --repo <OWNER/REPO>
```

### Summarize Changes

```bash
bitrise ai-reviewer summarize --code-review github --branch master --pr <PR_NUMBER> --repo <OWNER/REPO> 
```

### Commands

- `summarize`: Generate a concise summary of code changes
- `version`: Display the version information

### Flags

- `--pr`: The ID of the pull request to review
- `--repo`: The GitHub repository in the format 'owner/repo'
- `--branch`: Branch to review instead of a pull request
- `--code-review`: Code review provider (e.g., 'github')
- `--language`, `-l`: Language for AI responses (e.g., 'en-US', 'es-ES', 'fr-FR')
- `--profile`: Get the response in a more `chill`, or `assertive` format
- `--tone`: Tone to finetune the character and tone for the response

## Response Format

The AI reviewer provides structured feedback including:

- **Summary**: High-level overview of changes
- **Walkthrough**: Table of files and their change descriptions
- **Line Feedback**: Specific issues found in individual lines of code
- **Haiku**: A whimsical haiku summarizing the changes

## Development

### Prerequisites

- Go 1.24 or later
- Bitrise CLI installed
- GitHub API token for PR access
- OpenAI API key for AI analysis

### Building

```bash
go build -o bin/ai-reviewer
```

### Testing

```bash
go test ./...
```

### Installing locally

```bash
go build -o bin/ai-reviewer
bitrise plugin install --source .
```