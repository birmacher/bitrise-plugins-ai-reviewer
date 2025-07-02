package common

import "fmt"

const (
	ProfileChill     = "chill"
	ProfileAssertive = "assertive"
)

// Settings holds application-wide configuration settings
type Settings struct {
	Language string
	Profile  string
	Tone     string
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

// WithProfile sets the profile in Settings and returns the updated Settings
func (s Settings) WithProfile(profile string) Settings {
	if profile != ProfileChill && profile != ProfileAssertive {
		fmt.Println("Invalid profile setting, defaulting to 'chill'")
		profile = ProfileChill
	}
	s.Profile = profile
	return s
}

// GetProfile returns the current profile setting
func (s Settings) GetProfile() string {
	return s.Profile
}

// WithTone sets the tone in Settings and returns the updated Settings
func (s Settings) WithTone(tone string) Settings {
	s.Tone = tone
	return s
}

// GetTone returns the current tone setting
func (s Settings) GetTone() string {
	return s.Tone
}
