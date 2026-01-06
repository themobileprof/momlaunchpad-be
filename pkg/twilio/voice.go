package twilio

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// VoiceClient handles Twilio Voice operations
type VoiceClient struct {
	accountSID  string
	authToken   string
	phoneNumber string
}

// VoiceConfig holds Twilio Voice configuration
type VoiceConfig struct {
	AccountSID  string
	AuthToken   string
	PhoneNumber string
}

// NewVoiceClient creates a new Twilio Voice client
func NewVoiceClient(config VoiceConfig) *VoiceClient {
	return &VoiceClient{
		accountSID:  config.AccountSID,
		authToken:   config.AuthToken,
		phoneNumber: config.PhoneNumber,
	}
}

// ValidateRequest validates a Twilio webhook request signature
func (c *VoiceClient) ValidateRequest(url string, params map[string]string, signature string) bool {
	// Build data string as per Twilio's validation algorithm
	data := url

	// Sort keys alphabetically
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Append key-value pairs
	for _, k := range keys {
		data += k + params[k]
	}

	// Compute HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(c.authToken))
	mac.Write([]byte(data))
	expectedMAC := mac.Sum(nil)
	expectedSignature := base64.StdEncoding.EncodeToString(expectedMAC)

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// TwiMLResponse represents a TwiML response
type TwiMLResponse struct {
	builder strings.Builder
}

// NewTwiMLResponse creates a new TwiML response builder
func NewTwiMLResponse() *TwiMLResponse {
	t := &TwiMLResponse{}
	t.builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	t.builder.WriteString(`<Response>`)
	return t
}

// Say adds a Say verb to speak text
func (t *TwiMLResponse) Say(text, voice, language string) *TwiMLResponse {
	if voice == "" {
		voice = "Polly.Joanna" // Default to AWS Polly female voice
	}
	if language == "" {
		language = "en-US"
	}

	t.builder.WriteString(fmt.Sprintf(`<Say voice="%s" language="%s">%s</Say>`,
		voice, language, escapeXML(text)))
	return t
}

// Gather adds a Gather verb to collect speech input
func (t *TwiMLResponse) Gather(action, input, language string, timeout int) *TwiMLResponse {
	if input == "" {
		input = "speech"
	}
	if language == "" {
		language = "en-US"
	}
	if timeout == 0 {
		timeout = 5
	}

	t.builder.WriteString(fmt.Sprintf(
		`<Gather action="%s" input="%s" language="%s" timeout="%d" speechTimeout="auto">`,
		action, input, language, timeout))
	return t
}

// EndGather closes a Gather verb
func (t *TwiMLResponse) EndGather() *TwiMLResponse {
	t.builder.WriteString(`</Gather>`)
	return t
}

// Redirect adds a Redirect verb
func (t *TwiMLResponse) Redirect(url string) *TwiMLResponse {
	t.builder.WriteString(fmt.Sprintf(`<Redirect>%s</Redirect>`, escapeXML(url)))
	return t
}

// Hangup adds a Hangup verb
func (t *TwiMLResponse) Hangup() *TwiMLResponse {
	t.builder.WriteString(`<Hangup/>`)
	return t
}

// Pause adds a Pause verb
func (t *TwiMLResponse) Pause(length int) *TwiMLResponse {
	if length == 0 {
		length = 1
	}
	t.builder.WriteString(fmt.Sprintf(`<Pause length="%d"/>`, length))
	return t
}

// String returns the complete TwiML XML
func (t *TwiMLResponse) String() string {
	return t.builder.String() + `</Response>`
}

// CallStatus represents the status of a Twilio call
type CallStatus string

const (
	CallStatusQueued     CallStatus = "queued"
	CallStatusRinging    CallStatus = "ringing"
	CallStatusInProgress CallStatus = "in-progress"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusBusy       CallStatus = "busy"
	CallStatusFailed     CallStatus = "failed"
	CallStatusNoAnswer   CallStatus = "no-answer"
	CallStatusCanceled   CallStatus = "canceled"
)

// IncomingCallParams represents parameters from an incoming call webhook
type IncomingCallParams struct {
	CallSid       string
	AccountSid    string
	From          string
	To            string
	CallStatus    CallStatus
	Direction     string
	ForwardedFrom string
}

// GatherParams represents parameters from a Gather callback
type GatherParams struct {
	CallSid              string
	AccountSid           string
	SpeechResult         string
	Confidence           string
	UnstableSpeechResult string
}

// ParseIncomingCall parses URL values into IncomingCallParams
func ParseIncomingCall(values url.Values) IncomingCallParams {
	return IncomingCallParams{
		CallSid:       values.Get("CallSid"),
		AccountSid:    values.Get("AccountSid"),
		From:          values.Get("From"),
		To:            values.Get("To"),
		CallStatus:    CallStatus(values.Get("CallStatus")),
		Direction:     values.Get("Direction"),
		ForwardedFrom: values.Get("ForwardedFrom"),
	}
}

// ParseGather parses URL values into GatherParams
func ParseGather(values url.Values) GatherParams {
	return GatherParams{
		CallSid:              values.Get("CallSid"),
		AccountSid:           values.Get("AccountSid"),
		SpeechResult:         values.Get("SpeechResult"),
		Confidence:           values.Get("Confidence"),
		UnstableSpeechResult: values.Get("UnstableSpeechResult"),
	}
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// GetVoiceForLanguage returns appropriate Polly voice for language
func GetVoiceForLanguage(language string) string {
	voices := map[string]string{
		"en": "Polly.Joanna",
		"es": "Polly.Lupe",
		"fr": "Polly.Celine",
		"pt": "Polly.Vitoria",
		"de": "Polly.Vicki",
	}

	if voice, ok := voices[language]; ok {
		return voice
	}
	return "Polly.Joanna" // Default
}

// GetTwilioLanguageCode maps our language codes to Twilio's format
func GetTwilioLanguageCode(language string) string {
	codes := map[string]string{
		"en": "en-US",
		"es": "es-ES",
		"fr": "fr-FR",
		"pt": "pt-BR",
		"de": "de-DE",
	}

	if code, ok := codes[language]; ok {
		return code
	}
	return "en-US" // Default
}
