# Bitrise AI Reviewer Plugin

A Bitrise CLI plugin for reviewing code changes using AI.

## Overview

This plugin helps to automate code reviews by using AI to analyze pull requests and provide feedback, suggestions, and potential issue detection.

## Installation

```bash
bitrise plugin install --source https://github.com/birmacher/bitrise-plugins-ai-reviewer
```

Or you can install it from local source:

```bash
git clone https://github.com/birmacher/bitrise-plugins-ai-reviewer.git
cd bitrise-plugins-ai-reviewer
make install
```

## Usage

```
bitrise ai-reviewer review --pr <PR_URL>
```

### Commands

- `review`: Analyze a pull request and provide AI feedback

### Flags

- `--pr, -p`: The URL or ID of the pull request to review
- `--branch, -b`: Branch to review instead of a pull request

## Development

### Prerequisites

- Go 1.19 or later
- Bitrise CLI installed

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Installing locally

```bash
make install
```

## License

MIT
