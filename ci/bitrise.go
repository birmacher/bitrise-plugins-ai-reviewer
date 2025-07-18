package ci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
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

func GetBuildLog(appSlug, buildSlug string) (string, error) {
	position := 0
	isFinished := false
	foundTargetMessage := false
	targetLogMessage := "Found target message. Collecting a few more lines..."
	interval := 5

	outputFile, err := os.CreateTemp("", "bitrise_*.log")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary log file: %w", err)
	}
	defer outputFile.Close()

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
	os.Remove(outputFile.Name())

	return redactedLogs, nil
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

		if strings.Contains(line, stepHeaderStart()) && strings.Contains(lines[i+2], stepHeaderStart()) {
			headerStarted = true

			currentStepContent = []string{}
		}

		if headerStarted && strings.Contains(line, stepFooterStart()) {
			// Check if the build has failed
			if strings.Contains(lines[i+1], "31;1m") && strings.Contains(lines[i+3], "Issue tracker:") {
				logWithFailingSteps = append(logWithFailingSteps, currentStepContent...)
				logWithFailingSteps = append(logWithFailingSteps, lines[i:i+5]...)

				// Skipping the footer
				i += 5
			} else {
				// Header
				logWithFailingSteps = append(logWithFailingSteps, currentStepContent[0:9]...)
				logWithFailingSteps = append(logWithFailingSteps, "[successful step log truncated]")
				logWithFailingSteps = append(logWithFailingSteps, lines[i:i+3]...)

				// Skipping foother lines
				i += 3
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
