package language

import "sync"

const DefaultLanguage = "en"

// LanguageInfo contains information about a supported language
type LanguageInfo struct {
Code           string `json:"code"`
Name           string `json:"name"`
NativeName     string `json:"native_name"`
IsEnabled      bool   `json:"is_enabled"`
IsExperimental bool   `json:"is_experimental"`
}

// ValidationResult represents the result of language validation
type ValidationResult struct {
Code         string `json:"code"`
UsedFallback bool   `json:"used_fallback"`
}

// Manager handles language support and validation
type Manager struct {
languages map[string]*LanguageInfo
mu        sync.RWMutex
}

// NewManager creates a new language manager with default languages
func NewManager() *Manager {
return &Manager{
languages: map[string]*LanguageInfo{
"en": {
Code:           "en",
Name:           "English",
NativeName:     "English",
IsEnabled:      true,
IsExperimental: false,
},
"es": {
Code:           "es",
Name:           "Spanish",
NativeName:     "Español",
IsEnabled:      true,
IsExperimental: false,
},
},
}
}

// IsSupported checks if a language code is supported and enabled
func (m *Manager) IsSupported(code string) bool {
m.mu.RLock()
defer m.mu.RUnlock()

lang, exists := m.languages[code]
return exists && lang.IsEnabled
}

// Validate validates a language code and returns the validated code
// If the language is not supported, it falls back to the default language
func (m *Manager) Validate(code string) ValidationResult {
if m.IsSupported(code) {
return ValidationResult{
Code:         code,
UsedFallback: false,
}
}

// Fallback to default language
return ValidationResult{
Code:         DefaultLanguage,
UsedFallback: true,
}
}

// GetLanguageInfo returns information about a language
func (m *Manager) GetLanguageInfo(code string) (LanguageInfo, bool) {
m.mu.RLock()
defer m.mu.RUnlock()

lang, exists := m.languages[code]
if !exists {
return LanguageInfo{}, false
}

return *lang, true
}

// EnableLanguage enables a language
func (m *Manager) EnableLanguage(code string) {
m.mu.Lock()
defer m.mu.Unlock()

if lang, exists := m.languages[code]; exists {
lang.IsEnabled = true
}
}

// DisableLanguage disables a language (cannot disable default language)
func (m *Manager) DisableLanguage(code string) {
m.mu.Lock()
defer m.mu.Unlock()

// Cannot disable default language
if code == DefaultLanguage {
return
}

if lang, exists := m.languages[code]; exists {
lang.IsEnabled = false
}
}

// GetSupportedLanguages returns a list of all enabled languages
func (m *Manager) GetSupportedLanguages() []LanguageInfo {
m.mu.RLock()
defer m.mu.RUnlock()

var languages []LanguageInfo
for _, lang := range m.languages {
if lang.IsEnabled {
languages = append(languages, *lang)
}
}

return languages
}

// AddLanguage adds a new language (useful for admin operations)
func (m *Manager) AddLanguage(info LanguageInfo) {
m.mu.Lock()
defer m.mu.Unlock()

m.languages[info.Code] = &info
}
