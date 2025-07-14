package common

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestEncodeLLMKey(t *testing.T) {
	JSONResponse := `{
		"summary": "This is a test",
		"walkthrough": [
			{
			"files": "internal/service/code_push_manager.go, internal/service/code_push_manager_test.go",
			"summary": "Removed the large, monolithic CodePushManager implementation and its tests. This file previously contained all deployment, package, promote, and rollback logic."
			},
			{
			"files": "internal/service/create_deployment.go, internal/service/create_deployment_test.go, internal/service/delete_app_related_entities.go, internal/service/delete_app_related_entities_test.go, internal/service/delete_deployment.go, internal/service/delete_deployment_test.go, internal/service/delete_package.go, internal/service/delete_package_test.go, internal/service/get_deployment.go, internal/service/get_deployment_test.go, internal/service/get_package.go, internal/service/get_package_test.go, internal/service/list_deployments.go, internal/service/list_deployments_test.go, internal/service/list_packages.go, internal/service/list_packages_test.go, internal/service/promote.go, internal/service/promote_test.go, internal/service/rollback.go, internal/service/service_test.go, internal/service/start_release_package.go, internal/service/start_release_package_test.go, internal/service/update_deployment.go, internal/service/update_deployment_test.go, internal/service/update_package.go, internal/service/update_package_test.go",
			"summary": "Refactored the service layer into smaller, single-responsibility files. Each file now implements a specific part of the service logic (e.g., create, update, delete, promote, rollback, etc.), and each has its own dedicated test file. This improves maintainability and testability."
			}
		],
		"line-feedback": [
			{
				"file": "test.go",
				"content": "` + "```" + `	func example() {` + "```" + `",
				"suggestion": "` + "```" + `\tfunc example() {\n\t\tfmt.Println("This is a suggestion")` + "```" + `",
				"category": "test-category",
				"title": "Test Title"
			},
			{
				"file": "test.go",
				"content": "` + "```" + `	func example() {` + "```" + `",
				"suggestion": "` + "```" + `	func example() {
		fmt.Println("This is a suggestion")` + "```" + `",
				"category": "test-category",
				"title": "Test Title"
			}
		],
		"haiku": "> This is a haiku
> with multiple lines
> how great"
}`
	expectedLine := "CWZ1bmMgZXhhbXBsZSgpIHs="
	expectedSuggestion := "XHRmdW5jIGV4YW1wbGUoKSB7XG5cdFx0Zm10LlByaW50bG4oIlRoaXMgaXMgYSBzdWdnZXN0aW9uIik="
	expectedSuggestionMultiline := "CWZ1bmMgZXhhbXBsZSgpIHsKCQlmbXQuUHJpbnRsbigiVGhpcyBpcyBhIHN1Z2dlc3Rpb24iKQ=="
	expectedHaiku := "PiBUaGlzIGlzIGEgaGFpa3UKPiB3aXRoIG11bHRpcGxlIGxpbmVzCj4gaG93IGdyZWF0"

	encodedLine := EncodeLLMKey(JSONResponse, "content", true)
	encodedLine = EncodeLLMKey(encodedLine, "suggestion", true)
	encodedLine = EncodeLLMKey(encodedLine, "haiku", false)

	fmt.Println(encodedLine)

	summary := Summary{}
	if err := json.Unmarshal([]byte(encodedLine), &summary); err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	lineFeedback := LineLevelFeedback{}
	if err := json.Unmarshal([]byte(encodedLine), &lineFeedback); err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	if summary.Haiku != expectedHaiku {
		t.Errorf("Expected haiku to be %s, got %s", expectedHaiku, summary.Haiku)
	}

	if lineFeedback.Lines[0].Line != expectedLine {
		t.Errorf("Expected line to be %s, got %s", expectedLine, lineFeedback.Lines[0].Line)
	}

	if lineFeedback.Lines[0].Suggestion != expectedSuggestion {
		t.Errorf("Expected suggestion to be %s, got %s", expectedSuggestion, lineFeedback.Lines[0].Suggestion)
	}

	if lineFeedback.Lines[1].Line != expectedLine {
		t.Errorf("Expected line to be %s, got %s", expectedLine, lineFeedback.Lines[1].Line)
	}

	if lineFeedback.Lines[1].Suggestion != expectedSuggestionMultiline {
		t.Errorf("Expected suggestion to be %s, got %s", expectedSuggestionMultiline, lineFeedback.Lines[1].Suggestion)
	}
}

func TestDecodeLLMValue(t *testing.T) {
	// Test base64 encoded value
	encodedValue := "CWZ1bmMgZXhhbXBsZSgpIHsKCQlmbXQuUHJpbnRsbigiVGhpcyBpcyBhIHN1Z2dlc3Rpb24iKQ=="
	expectedValue := `	func example() {
		fmt.Println("This is a suggestion")`

	decodedValue, err := DecodeLLMValue(encodedValue)
	if err != nil {
		t.Errorf("Failed to decode value: %v", err)
	}

	if decodedValue != expectedValue {
		t.Errorf("Expected decoded value to be %s, got %s", expectedValue, decodedValue)
	}
	
	// Test non-base64 value (plain text)
	plainValue := "> New tools in the breeze\n> Codebase whispers, search, blame, fetch—\n> Review magic grows 🌱🤖"
	
	decodedPlainValue, err := DecodeLLMValue(plainValue)
	if err != nil {
		t.Errorf("Failed to handle non-base64 value: %v", err)
	}
	
	if decodedPlainValue != plainValue {
		t.Errorf("Expected plain text value to be returned as-is, got %s", decodedPlainValue)
	}
}
