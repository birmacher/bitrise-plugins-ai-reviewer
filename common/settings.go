package common

// Settings holds application-wide configuration settings
type Settings struct {
	// Language represents the language code (e.g., "en-US", "es-ES")
	Language string
}

// WithLanguage sets the language in Settings and returns the updated Settings
func (s Settings) WithLanguage(language string) Settings {
	s.Language = language
	return s
}

// GetLanguage returns the current language setting
func (s Settings) GetLanguage() string {
	return s.Language
}
