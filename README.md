# Bitrise AI Reviewer Plugin

A Bitrise CLI plugin for reviewing and summarizing code changes using AI.

## Overview

This plugin helps to automate code reviews by using AI to analyze pull requests and provide feedback, suggestions, and potential issue detection. It uses the OpenAI API to generate insights about your code changes, with structured responses focusing on code quality, potential bugs, and performance issues.

## Features

- **PR Review**: Analyze GitHub pull requests for potential issues
- **Code Summarization**: Generate concise summaries of code changes
- **Line-by-Line Feedback**: Get specific feedback on individual code lines
- **Integration with GitHub**: Automatically fetch PR details and provide feedback

## Installation

```bash
bitrise plugin install --source https://github.com/bitrise-io/bitrise-plugins-ai-reviewer
```

Or you can install it from local source:

```bash
git clone https://github.com/bitrise-io/bitrise-plugins-ai-reviewer.git
cd bitrise-plugins-ai-reviewer
go build -o bin/ai-reviewer
bitrise plugin install --source .
```

## Configuration

Set up your environment with the necessary API tokens:

```bash
export GITHUB_TOKEN=your_github_personal_access_token
export OPENAI_API_KEY=your_openai_api_key
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