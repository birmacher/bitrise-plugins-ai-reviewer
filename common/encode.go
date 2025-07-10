package common

import (
	"encoding/base64"
	"fmt"
	"regexp"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
)

func EncodeLLMKey(jsonStr, key string) string {
	// This regex finds the key and captures its string value, handling escaped quotes.
	pattern := fmt.Sprintf(`"%s":\s*"((?:\\.|[^"\\])*)"`, key)
	re := regexp.MustCompile(pattern)

	return re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			// Should not happen if the main regex matched, but as a safeguard.
			return match
		}

		// The captured value is in submatches[1].
		originalValue := submatches[1]

		// Base64 encode the original value.
		encodedValue := base64.StdEncoding.EncodeToString([]byte(originalValue))

		// Return the key with the new Base64 encoded value.
		return fmt.Sprintf(`"%s": "%s"`, key, encodedValue)
	})
}

func DecodeLLMValue(value string) (string, error) {
	decodedLine, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		logger.Warnf("Error decoding base64 value: %v", err)
		return "", err
	}
	return string(decodedLine), nil
}
