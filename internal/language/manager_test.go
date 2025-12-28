package language

import (
"testing"
)

func TestManager_IsSupported(t *testing.T) {
tests := []struct {
name       string
langCode   string
wantResult bool
}{
{
name:       "English is supported",
langCode:   "en",
wantResult: true,
},
{
name:       "Spanish is supported",
langCode:   "es",
wantResult: true,
},
{
name:       "French is not supported",
langCode:   "fr",
wantResult: false,
},
{
name:       "Invalid code is not supported",
langCode:   "invalid",
wantResult: false,
},
{
name:       "Empty code is not supported",
langCode:   "",
wantResult: false,
},
}

manager := NewManager()
for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := manager.IsSupported(tt.langCode)
if result != tt.wantResult {
t.Errorf("IsSupported(%s) = %v, want %v", tt.langCode, result, tt.wantResult)
}
})
}
}

func TestManager_Validate(t *testing.T) {
tests := []struct {
name       string
langCode   string
wantCode   string
wantFallback bool
}{
{
name:       "Valid English returns en",
langCode:   "en",
wantCode:   "en",
wantFallback: false,
},
{
name:       "Valid Spanish returns es",
langCode:   "es",
wantCode:   "es",
wantFallback: false,
},
{
name:       "Unsupported language falls back to English",
langCode:   "fr",
wantCode:   "en",
wantFallback: true,
},
{
name:       "Empty language falls back to English",
langCode:   "",
wantCode:   "en",
wantFallback: true,
},
{
name:       "Invalid language falls back to English",
langCode:   "invalid",
wantCode:   "en",
wantFallback: true,
},
}

manager := NewManager()
for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := manager.Validate(tt.langCode)
if result.Code != tt.wantCode {
t.Errorf("Validate(%s).Code = %s, want %s", tt.langCode, result.Code, tt.wantCode)
}
if result.UsedFallback != tt.wantFallback {
t.Errorf("Validate(%s).UsedFallback = %v, want %v", tt.langCode, result.UsedFallback, tt.wantFallback)
}
})
}
}

func TestManager_GetLanguageInfo(t *testing.T) {
tests := []struct {
name       string
langCode   string
wantName   string
wantNative string
wantFound  bool
}{
{
name:       "English info",
langCode:   "en",
wantName:   "English",
wantNative: "English",
wantFound:  true,
},
{
name:       "Spanish info",
langCode:   "es",
wantName:   "Spanish",
wantNative: "Español",
wantFound:  true,
},
{
name:       "Unsupported language returns not found",
langCode:   "fr",
wantName:   "",
wantNative: "",
wantFound:  false,
},
}

manager := NewManager()
for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
info, found := manager.GetLanguageInfo(tt.langCode)
if found != tt.wantFound {
t.Errorf("GetLanguageInfo(%s) found = %v, want %v", tt.langCode, found, tt.wantFound)
}
if found {
if info.Name != tt.wantName {
t.Errorf("GetLanguageInfo(%s).Name = %s, want %s", tt.langCode, info.Name, tt.wantName)
}
if info.NativeName != tt.wantNative {
t.Errorf("GetLanguageInfo(%s).NativeName = %s, want %s", tt.langCode, info.NativeName, tt.wantNative)
}
}
})
}
}

func TestManager_EnableDisableLanguage(t *testing.T) {
manager := NewManager()

// Initially Spanish should be enabled
if !manager.IsSupported("es") {
t.Error("Spanish should be enabled initially")
}

// Disable Spanish
manager.DisableLanguage("es")
if manager.IsSupported("es") {
t.Error("Spanish should be disabled after DisableLanguage")
}

// Re-enable Spanish
manager.EnableLanguage("es")
if !manager.IsSupported("es") {
t.Error("Spanish should be enabled after EnableLanguage")
}

// Cannot disable default language
manager.DisableLanguage("en")
if !manager.IsSupported("en") {
t.Error("English (default) should always be enabled")
}
}

func TestManager_GetSupportedLanguages(t *testing.T) {
manager := NewManager()

langs := manager.GetSupportedLanguages()

if len(langs) < 2 {
t.Errorf("Expected at least 2 supported languages, got %d", len(langs))
}

// Check that English is in the list
foundEN := false
for _, lang := range langs {
if lang.Code == "en" {
foundEN = true
break
}
}
if !foundEN {
t.Error("English should be in supported languages list")
}
}
