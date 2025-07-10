package common

import (
	"os"
	"testing"
)

func TestWithDefaultSettings(t *testing.T) {
	settings := WithDefaultSettings()

	// Test default language
	if settings.Language != "en-US" {
		t.Errorf("Expected default language to be en-US, got %s", settings.Language)
	}

	// Test default reviews settings
	if !settings.Reviews.Summary {
		t.Error("Expected default Summary to be true")
	}

	if !settings.Reviews.Walkthrough {
		t.Error("Expected default Walkthrough to be true")
	}

	if !settings.Reviews.CollapseWalkthrough {
		t.Error("Expected default CollapseWalkthrough to be true")
	}

	if !settings.Reviews.Haiku {
		t.Error("Expected default Haiku to be true")
	}

	if settings.Reviews.Profile != ProfileChill {
		t.Errorf("Expected default Profile to be %s, got %s", ProfileChill, settings.Reviews.Profile)
	}

	// Test default path filters and instructions
	if settings.Reviews.PathFilters != "" {
		t.Errorf("Expected empty PathFilters by default, got %s", settings.Reviews.PathFilters)
	}

	if settings.Reviews.PathInstructions != "" {
		t.Errorf("Expected empty PathInstructions by default, got %s", settings.Reviews.PathInstructions)
	}

	if settings.Tone != "" {
		t.Errorf("Expected empty Tone by default, got %s", settings.Tone)
	}
}

func TestWithYamlFile_ValidFile(t *testing.T) {
	// Create a temporary config file
	configContent := `language: fr-FR
tone_instructions: friendly
reviews:
  profile: assertive
  summary: false
  walkthrough: false
  collapse_walkthrough: false
  haiku: false
  path_filters: "*.go,*.js"
  path_instructions: "Review Go files carefully"
`
	tempDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to temp directory to create the config file
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(cwd) // Restore original directory when done

	if err := os.WriteFile("review.bitrise.yml", []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test reading from the config file
	settings := WithYamlFile()

	// Verify settings were loaded from file
	expectedSettings := Settings{
		Language: "fr-FR",
		Tone:     "friendly",
		Reviews: Reviews{
			Profile:             ProfileAssertive,
			Summary:             false,
			Walkthrough:         false,
			CollapseWalkthrough: false,
			Haiku:               false,
			PathFilters:         "*.go,*.js",
			PathInstructions:    "Review Go files carefully",
		},
	}

	if settings.Language != expectedSettings.Language {
		t.Errorf("Expected language %s, got %s", expectedSettings.Language, settings.Language)
	}

	if settings.Tone != expectedSettings.Tone {
		t.Errorf("Expected tone %s, got %s", expectedSettings.Tone, settings.Tone)
	}

	if settings.Reviews.Profile != expectedSettings.Reviews.Profile {
		t.Errorf("Expected profile %s, got %s", expectedSettings.Reviews.Profile, settings.Reviews.Profile)
	}

	if settings.Reviews.Summary != expectedSettings.Reviews.Summary {
		t.Errorf("Expected summary %v, got %v", expectedSettings.Reviews.Summary, settings.Reviews.Summary)
	}

	if settings.Reviews.PathFilters != expectedSettings.Reviews.PathFilters {
		t.Errorf("Expected path filters %s, got %s", expectedSettings.Reviews.PathFilters, settings.Reviews.PathFilters)
	}

	if settings.Reviews.PathInstructions != expectedSettings.Reviews.PathInstructions {
		t.Errorf("Expected path instructions %s, got %s", expectedSettings.Reviews.PathInstructions, settings.Reviews.PathInstructions)
	}
}

func TestWithYamlFileInSubdirectory_ValidFile(t *testing.T) {
	// Create a temporary config file
	configContent := `language: fr-FR
tone_instructions: friendly
reviews:
  profile: assertive
  summary: false
  walkthrough: false
  collapse_walkthrough: false
  haiku: false
  path_filters: "*.go,*.js"
  path_instructions: "Review Go files carefully"
`
	tempDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	if err := os.MkdirAll(tempDir+"/subdir", 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Change to temp directory to create the config file
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(cwd) // Restore original directory when done

	if err := os.WriteFile("subdir/review.bitrise.yml", []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test reading from the config file
	settings := WithYamlFile()

	// Verify settings were loaded from file
	expectedSettings := Settings{
		Language: "fr-FR",
		Tone:     "friendly",
		Reviews: Reviews{
			Profile:             ProfileAssertive,
			Summary:             false,
			Walkthrough:         false,
			CollapseWalkthrough: false,
			Haiku:               false,
			PathFilters:         "*.go,*.js",
			PathInstructions:    "Review Go files carefully",
		},
	}

	if settings.Language != expectedSettings.Language {
		t.Errorf("Expected language %s, got %s", expectedSettings.Language, settings.Language)
	}

	if settings.Tone != expectedSettings.Tone {
		t.Errorf("Expected tone %s, got %s", expectedSettings.Tone, settings.Tone)
	}

	if settings.Reviews.Profile != expectedSettings.Reviews.Profile {
		t.Errorf("Expected profile %s, got %s", expectedSettings.Reviews.Profile, settings.Reviews.Profile)
	}

	if settings.Reviews.Summary != expectedSettings.Reviews.Summary {
		t.Errorf("Expected summary %v, got %v", expectedSettings.Reviews.Summary, settings.Reviews.Summary)
	}

	if settings.Reviews.PathFilters != expectedSettings.Reviews.PathFilters {
		t.Errorf("Expected path filters %s, got %s", expectedSettings.Reviews.PathFilters, settings.Reviews.PathFilters)
	}

	if settings.Reviews.PathInstructions != expectedSettings.Reviews.PathInstructions {
		t.Errorf("Expected path instructions %s, got %s", expectedSettings.Reviews.PathInstructions, settings.Reviews.PathInstructions)
	}
}

func TestWithYamlFile_PartialYam(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to temp directory to create the config file
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(cwd) // Restore original directory when done

	// Create invalid YAML content
	invalidContent := `language: fr-FR
reviews:
  profile: assertive
  this-is-invalid: yaml
`

	if err := os.WriteFile("review.bitrise.yml", []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	// Test that default settings are returned when config file has invalid format
	settings := WithYamlFile()

	// Should still get some values from the partially valid YAML
	// but missing or invalid parts should use defaults
	if settings.Language != "fr-FR" {
		t.Errorf("Expected language fr-FR (from partial parsing), got %s", settings.Language)
	}

	if settings.Reviews.Profile != ProfileAssertive {
		t.Errorf("Expected language fr-FR (from partial parsing), got %s", settings.Language)
	}

	// Other values should be default
	expectedSettings := WithDefaultSettings()

	if settings.Tone != expectedSettings.Tone {
		t.Errorf("Expected tone %s, got %s", expectedSettings.Tone, settings.Tone)
	}

	if settings.Reviews.Summary != expectedSettings.Reviews.Summary {
		t.Errorf("Expected summary %v, got %v", expectedSettings.Reviews.Summary, settings.Reviews.Summary)
	}

	if settings.Reviews.PathFilters != expectedSettings.Reviews.PathFilters {
		t.Errorf("Expected path filters %s, got %s", expectedSettings.Reviews.PathFilters, settings.Reviews.PathFilters)
	}

	if settings.Reviews.PathInstructions != expectedSettings.Reviews.PathInstructions {
		t.Errorf("Expected path instructions %s, got %s", expectedSettings.Reviews.PathInstructions, settings.Reviews.PathInstructions)
	}
}

func TestWithYamlFile_InvalidYaml(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to temp directory to create the config file
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(cwd) // Restore original directory when done

	// Create invalid YAML content
	invalidContent := `language: fr-FR
reviews:
  profile: assertive
  this-is-invalid-yaml
`

	if err := os.WriteFile("review.bitrise.yml", []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	// Test that default settings are returned when config file has invalid format
	settings := WithYamlFile()

	expectedSettings := WithDefaultSettings()

	if settings.Language != expectedSettings.Language {
		t.Errorf("Expected language %s, got %s", expectedSettings.Language, settings.Language)
	}

	if settings.Tone != expectedSettings.Tone {
		t.Errorf("Expected tone %s, got %s", expectedSettings.Tone, settings.Tone)
	}

	if settings.Reviews.Profile != expectedSettings.Reviews.Profile {
		t.Errorf("Expected profile %s, got %s", expectedSettings.Reviews.Profile, settings.Reviews.Profile)
	}

	if settings.Reviews.Summary != expectedSettings.Reviews.Summary {
		t.Errorf("Expected summary %v, got %v", expectedSettings.Reviews.Summary, settings.Reviews.Summary)
	}

	if settings.Reviews.PathFilters != expectedSettings.Reviews.PathFilters {
		t.Errorf("Expected path filters %s, got %s", expectedSettings.Reviews.PathFilters, settings.Reviews.PathFilters)
	}

	if settings.Reviews.PathInstructions != expectedSettings.Reviews.PathInstructions {
		t.Errorf("Expected path instructions %s, got %s", expectedSettings.Reviews.PathInstructions, settings.Reviews.PathInstructions)
	}
}

func TestWithYamlFile_EmptyFile(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to temp directory to create the config file
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(cwd) // Restore original directory when done

	// Create empty file
	if err := os.WriteFile("review.bitrise.yml", []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty config file: %v", err)
	}

	// Test that default settings are returned when config file is empty
	settings := WithYamlFile()
	expectedSettings := WithDefaultSettings()

	if settings.Language != expectedSettings.Language {
		t.Errorf("Expected language %s, got %s", expectedSettings.Language, settings.Language)
	}

	if settings.Tone != expectedSettings.Tone {
		t.Errorf("Expected tone %s, got %s", expectedSettings.Tone, settings.Tone)
	}

	if settings.Reviews.Profile != expectedSettings.Reviews.Profile {
		t.Errorf("Expected profile %s, got %s", expectedSettings.Reviews.Profile, settings.Reviews.Profile)
	}

	if settings.Reviews.Summary != expectedSettings.Reviews.Summary {
		t.Errorf("Expected summary %v, got %v", expectedSettings.Reviews.Summary, settings.Reviews.Summary)
	}

	if settings.Reviews.PathFilters != expectedSettings.Reviews.PathFilters {
		t.Errorf("Expected path filters %s, got %s", expectedSettings.Reviews.PathFilters, settings.Reviews.PathFilters)
	}

	if settings.Reviews.PathInstructions != expectedSettings.Reviews.PathInstructions {
		t.Errorf("Expected path instructions %s, got %s", expectedSettings.Reviews.PathInstructions, settings.Reviews.PathInstructions)
	}
}

func TestWithYamlFile_PermissionDenied(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to temp directory to create the config file
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(cwd) // Restore original directory when done

	// Create a file with normal permissions first
	if err := os.WriteFile("review.bitrise.yml", []byte("language: fr-FR"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Skip this test on Windows as permissions work differently
	if os.Getenv("GOOS") != "windows" {
		// Change permissions to make it unreadable
		if err := os.Chmod("review.bitrise.yml", 0000); err != nil {
			t.Fatalf("Failed to change file permissions: %v", err)
		}

		// Test that default settings are returned when config file is unreadable
		settings := WithYamlFile()
		expectedSettings := WithDefaultSettings()

		if settings.Language != expectedSettings.Language {
			t.Errorf("Expected language %s, got %s", expectedSettings.Language, settings.Language)
		}

		if settings.Tone != expectedSettings.Tone {
			t.Errorf("Expected tone %s, got %s", expectedSettings.Tone, settings.Tone)
		}

		if settings.Reviews.Profile != expectedSettings.Reviews.Profile {
			t.Errorf("Expected profile %s, got %s", expectedSettings.Reviews.Profile, settings.Reviews.Profile)
		}

		if settings.Reviews.Summary != expectedSettings.Reviews.Summary {
			t.Errorf("Expected summary %v, got %v", expectedSettings.Reviews.Summary, settings.Reviews.Summary)
		}

		if settings.Reviews.PathFilters != expectedSettings.Reviews.PathFilters {
			t.Errorf("Expected path filters %s, got %s", expectedSettings.Reviews.PathFilters, settings.Reviews.PathFilters)
		}

		if settings.Reviews.PathInstructions != expectedSettings.Reviews.PathInstructions {
			t.Errorf("Expected path instructions %s, got %s", expectedSettings.Reviews.PathInstructions, settings.Reviews.PathInstructions)
		}

		// Restore permissions so the file can be deleted
		os.Chmod("review.bitrise.yml", 0644)
	}
}

func TestConstantValues(t *testing.T) {
	// Test constant values are as expected
	if ProfileChill != "chill" {
		t.Errorf("Expected ProfileChill constant to be 'chill', got %s", ProfileChill)
	}

	if ProfileAssertive != "assertive" {
		t.Errorf("Expected ProfileAssertive constant to be 'assertive', got %s", ProfileAssertive)
	}
}
