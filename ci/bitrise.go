package ci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
)

type BitriseLogResponse struct {
	LogChunks           []LogChunk `json:"log_chunks"`
	NextBeforeTimestamp string     `json:"next_before_timestamp"`
	NextAfterTimestamp  string     `json:"next_after_timestamp"`
	IsArchived          bool       `json:"is_archived"`
	ExpiringRawLogURL   string     `json:"expiring_raw_log_url,omitempty"`
}

type LogChunk struct {
	Chunk    string `json:"chunk"`
	Position int    `json:"position"`
}

func getToken() (string, error) {
	token := os.Getenv("BITRISE_TOKEN")
	if token == "" {
		return "", fmt.Errorf("BITRISE_TOKEN environment variable is not set")
	}
	return token, nil
}

func GetAppID() (string, error) {
	appSlug := os.Getenv("BITRISE_APP_SLUG")
	if appSlug == "" {
		return "", fmt.Errorf("BITRISE_APP_SLUG environment variable is not set")
	}
	return appSlug, nil
}

func GetBuildID() (string, error) {
	buildID := os.Getenv("BITRISE_BUILD_SLUG")
	if buildID == "" {
		return "", fmt.Errorf("BITRISE_BUILD_SLUG environment variable is not set")
	}
	return buildID, nil
}

func GetCommitHash() (string, error) {
	commitHash := os.Getenv("BITRISE_GIT_COMMIT")
	if commitHash == "" {
		logger.Warn("BITRISE_GIT_COMMIT environment variable is not set, using HEAD commit hash")
		commitHash = "HEAD"
	}
	return commitHash, nil
}

func GetBuildLog() (string, error) {
	position := 0
	isFinished := false
	foundTargetMessage := false
	targetLogMessage := "Running AI build summary..."
	interval := 5

	appSlug, err := GetAppID()
	if err != nil {
		return "", fmt.Errorf("failed to get app ID: %w", err)
	}
	buildSlug, err := GetBuildID()
	if err != nil {
		return "", fmt.Errorf("failed to get build ID: %w", err)
	}

	outputFile, err := os.CreateTemp("", "bitrise_*.log")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary log file: %w", err)
	}
	defer outputFile.Close()
	defer os.Remove(outputFile.Name())

	for {
		logResponse, err := fetchLogChunk(appSlug, buildSlug, position)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching logs: %v\n", err)
			os.Exit(1)
		}

		// Process each log chunk
		if len(logResponse.LogChunks) > 0 {
			for _, chunk := range logResponse.LogChunks {
				if chunk.Chunk != "" {
					appendChunksToFile(outputFile.Name(), []string{chunk.Chunk})
				}

				// Update the last position to the highest position we've seen
				if chunk.Position > position {
					position = chunk.Position
				}

				if strings.Contains(chunk.Chunk, targetLogMessage) {
					// Just found the target
					foundTargetMessage = true
					fmt.Println("\nFound target message. Collecting a few more lines...")
				}
			}
		}
		// If the log is archived, we can consider it finished
		isFinished = logResponse.IsArchived

		// If build is finished, exit the loop
		if isFinished || foundTargetMessage {
			fmt.Printf("\nLog collection finished.")
			break
		}

		// Wait before polling again
		time.Sleep(time.Duration(interval) * time.Second)
	}

	redactedLogs, err := readLogFile(outputFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	return redactedLogs, nil
}

func PostBuildSummary(summary, suggestion string) error {
	if err := installAnnotationPlugin(); err != nil {
		return err
	}

	body := `## AI Build Summary

### Error details
` + summary

	if len(suggestion) > 0 {
		body += `
		
### Suggested fix
` + suggestion
	}

	logger.Debug("Posting build summary:", body)
	cmd := exec.Command("bitrise", ":annotations", "annotate", body, "--style", "info", "--context", "ai-summary")

	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		logger.Debug("Annotations failed:", output.String())
		return fmt.Errorf("failed to post build summary: %w", err)
	}

	return nil
}

func installAnnotationPlugin() error {
	// Try to list the annotation plugin
	cmd := exec.Command("bitrise", "plugins", ":annotations")
	if err := cmd.Run(); err != nil {
		// If not found, install it
		installCmd := exec.Command("bitrise", "plugin", "install", "https://github.com/bitrise-io/bitrise-plugins-annotations.git")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install annotations plugin: %w", err)
		}
	}
	return nil
}

func fetchLogChunk(appSlug, buildSlug string, position int) (BitriseLogResponse, error) {
	url := fmt.Sprintf("https://api.bitrise.io/v0.1/apps/%s/builds/%s/log", appSlug, buildSlug)

	// Add position parameter if not starting from the beginning
	if position > 0 {
		url = fmt.Sprintf("%s?from=%d", url, position)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return BitriseLogResponse{}, err
	}

	// Add authorization header
	token, err := getToken()
	if err != nil {
		return BitriseLogResponse{}, err
	}
	req.Header.Add("Authorization", "token "+token)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return BitriseLogResponse{}, err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return BitriseLogResponse{}, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	// Parse the response
	var logChunk BitriseLogResponse
	err = json.NewDecoder(resp.Body).Decode(&logChunk)
	if err != nil {
		return BitriseLogResponse{}, err
	}

	return logChunk, nil
}

func appendChunksToFile(filePath string, chunks []string) error {
	// Open file for appending (create if doesn't exist)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write each chunk
	for _, chunk := range chunks {
		if _, err := file.WriteString(chunk); err != nil {
			return err
		}
	}

	return nil
}

func readLogFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	logWithFailingSteps := []string{}
	currentStepContent := []string{}
	headerStarted := false

	lines := strings.Split(string(content), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if len(lines) > i+2 && strings.Contains(line, stepHeaderStart()) && strings.Contains(lines[i+2], stepHeaderStart()) {
			headerStarted = true

			currentStepContent = []string{}
		}

		if headerStarted && strings.Contains(line, stepFooterStart()) {
			headerLen := min(len(currentStepContent), 9)
			logWithFailingSteps = append(logWithFailingSteps, currentStepContent[0:headerLen]...)

			// Check if the build has failed
			if len(lines) > i+5 && strings.Contains(lines[i+1], "31;1m") && strings.Contains(lines[i+3], "Issue tracker:") {
				logWithFailingSteps = append(logWithFailingSteps, currentStepContent[headerLen:]...)
				logWithFailingSteps = append(logWithFailingSteps, lines[i:i+5]...)

				// Skipping the footer
				i += 5
			} else {
				// Header
				footerLen := 3
				if len(lines) < i+3 {
					footerLen = len(lines) - i
				}
				logWithFailingSteps = append(logWithFailingSteps, "[successful step log truncated]")
				logWithFailingSteps = append(logWithFailingSteps, lines[i:i+footerLen-1]...)

				// Skipping footer lines
				i += footerLen
			}

			headerStarted = false
		}

		// If we are not in a step, add the lines
		if !headerStarted {
			logWithFailingSteps = append(logWithFailingSteps, line)
		}

		if headerStarted {
			currentStepContent = append(currentStepContent, line)
		}
	}

	return strings.Join(logWithFailingSteps, "\n"), nil
}

func stepHeaderStart() string {
	return "+------------------------------------------------------------------------------+"
}

func stepFooterStart() string {
	return `+---+---------------------------------------------------------------+----------+`
}
