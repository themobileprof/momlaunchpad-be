package twilio

import (
	"net/url"
	"strings"
	"testing"
)

func TestNewTwiMLResponse(t *testing.T) {
	twiml := NewTwiMLResponse().
		Say("Hello, welcome to MomLaunchpad", "Polly.Joanna", "en-US").
		String()

	if !strings.Contains(twiml, "<Response>") {
		t.Error("Expected Response tag")
	}
	if !strings.Contains(twiml, "Hello, welcome to MomLaunchpad") {
		t.Error("Expected message text")
	}
	if !strings.Contains(twiml, "</Response>") {
		t.Error("Expected closing Response tag")
	}
}

func TestTwiMLGather(t *testing.T) {
	twiml := NewTwiMLResponse().
		Gather("/voice/process", "speech", "en-US", 5).
		Say("Please ask your question", "", "").
		EndGather().
		String()

	if !strings.Contains(twiml, "<Gather") {
		t.Error("Expected Gather tag")
	}
	if !strings.Contains(twiml, `action="/voice/process"`) {
		t.Error("Expected action attribute")
	}
	if !strings.Contains(twiml, "</Gather>") {
		t.Error("Expected closing Gather tag")
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello & goodbye", "Hello &amp; goodbye"},
		{"<tag>", "&lt;tag&gt;"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"normal text", "normal text"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeXML(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseIncomingCall(t *testing.T) {
	values := url.Values{
		"CallSid":    []string{"CA123"},
		"From":       []string{"+1234567890"},
		"To":         []string{"+0987654321"},
		"CallStatus": []string{"in-progress"},
	}

	params := ParseIncomingCall(values)

	if params.CallSid != "CA123" {
		t.Errorf("got CallSid %s, want CA123", params.CallSid)
	}
	if params.From != "+1234567890" {
		t.Errorf("got From %s, want +1234567890", params.From)
	}
	if params.CallStatus != CallStatusInProgress {
		t.Errorf("got CallStatus %s, want in-progress", params.CallStatus)
	}
}

func TestParseGather(t *testing.T) {
	values := url.Values{
		"CallSid":      []string{"CA123"},
		"SpeechResult": []string{"When will my baby kick"},
		"Confidence":   []string{"0.95"},
	}

	params := ParseGather(values)

	if params.CallSid != "CA123" {
		t.Errorf("got CallSid %s, want CA123", params.CallSid)
	}
	if params.SpeechResult != "When will my baby kick" {
		t.Errorf("got SpeechResult %s", params.SpeechResult)
	}
	if params.Confidence != "0.95" {
		t.Errorf("got Confidence %s, want 0.95", params.Confidence)
	}
}

func TestGetVoiceForLanguage(t *testing.T) {
	tests := []struct {
		lang     string
		expected string
	}{
		{"en", "Polly.Joanna"},
		{"es", "Polly.Lupe"},
		{"fr", "Polly.Celine"},
		{"unknown", "Polly.Joanna"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			voice := GetVoiceForLanguage(tt.lang)
			if voice != tt.expected {
				t.Errorf("got %s, want %s", voice, tt.expected)
			}
		})
	}
}
