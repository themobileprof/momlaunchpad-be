package fallback

import (
	"github.com/themobileprof/momlaunchpad-be/internal/classifier"
)

// Response represents a fallback response
type Response struct {
	Content string
	Action  string // "retry", "contact_support", "emergency"
}

var (
	// English fallbacks
	englishFallbacks = map[classifier.Intent]Response{
		classifier.IntentSymptom: {
			Content: "I'm having trouble processing your message right now. If you're experiencing severe symptoms like bleeding, severe pain, or other concerning signs, please contact your healthcare provider immediately or call emergency services.",
			Action:  "emergency",
		},
		classifier.IntentPregnancyQ: {
			Content: "I'm having a brief connection issue. Let me try again in a moment. In the meantime, if your question is urgent, please reach out to your healthcare provider.",
			Action:  "retry",
		},
		classifier.IntentScheduling: {
			Content: "I'm having trouble right now, but your calendar is still accessible. You can add reminders manually while I get back online.",
			Action:  "retry",
		},
		classifier.IntentSmallTalk: {
			Content: "I'm here! Having a small technical hiccup. How can I help you today?",
			Action:  "retry",
		},
		classifier.IntentUnclear: {
			Content: "I'm having trouble understanding right now. Could you try rephrasing your question?",
			Action:  "retry",
		},
	}

	// Spanish fallbacks
	spanishFallbacks = map[classifier.Intent]Response{
		classifier.IntentSymptom: {
			Content: "Estoy teniendo problemas para procesar tu mensaje ahora. Si estás experimentando síntomas graves como sangrado, dolor severo u otras señales preocupantes, contacta a tu proveedor de salud inmediatamente o llama a servicios de emergencia.",
			Action:  "emergency",
		},
		classifier.IntentPregnancyQ: {
			Content: "Tengo un problema de conexión breve. Déjame intentar de nuevo en un momento. Mientras tanto, si tu pregunta es urgente, comunícate con tu proveedor de salud.",
			Action:  "retry",
		},
		classifier.IntentScheduling: {
			Content: "Estoy teniendo problemas ahora, pero tu calendario sigue accesible. Puedes agregar recordatorios manualmente mientras vuelvo en línea.",
			Action:  "retry",
		},
		classifier.IntentSmallTalk: {
			Content: "¡Estoy aquí! Teniendo un pequeño problema técnico. ¿Cómo puedo ayudarte hoy?",
			Action:  "retry",
		},
		classifier.IntentUnclear: {
			Content: "Estoy teniendo problemas para entender ahora. ¿Podrías reformular tu pregunta?",
			Action:  "retry",
		},
	}

	// Timeout-specific fallbacks
	timeoutFallbacks = map[string]Response{
		"en": {
			Content: "I'm taking longer than usual to respond. This might be a temporary issue. If your question is urgent, please contact your healthcare provider.",
			Action:  "retry",
		},
		"es": {
			Content: "Estoy tardando más de lo habitual en responder. Esto podría ser un problema temporal. Si tu pregunta es urgente, contacta a tu proveedor de salud.",
			Action:  "retry",
		},
	}

	// Circuit breaker open fallbacks
	circuitOpenFallbacks = map[string]Response{
		"en": {
			Content: "I'm temporarily unavailable due to technical difficulties. I'll be back shortly. For urgent matters, please contact your healthcare provider directly.",
			Action:  "contact_support",
		},
		"es": {
			Content: "Estoy temporalmente no disponible debido a dificultades técnicas. Volveré pronto. Para asuntos urgentes, contacta directamente a tu proveedor de salud.",
			Action:  "contact_support",
		},
	}
)

// GetFallbackResponse returns an appropriate fallback response
func GetFallbackResponse(intent classifier.Intent, language string) Response {
	var fallbacks map[classifier.Intent]Response

	switch language {
	case "es":
		fallbacks = spanishFallbacks
	default:
		fallbacks = englishFallbacks
	}

	if response, ok := fallbacks[intent]; ok {
		return response
	}

	// Default fallback
	if language == "es" {
		return Response{
			Content: "Lo siento, estoy teniendo problemas técnicos. Por favor intenta de nuevo.",
			Action:  "retry",
		}
	}

	return Response{
		Content: "I'm sorry, I'm having technical difficulties. Please try again.",
		Action:  "retry",
	}
}

// GetTimeoutResponse returns a timeout-specific fallback
func GetTimeoutResponse(language string) Response {
	if response, ok := timeoutFallbacks[language]; ok {
		return response
	}
	return timeoutFallbacks["en"]
}

// GetCircuitOpenResponse returns a circuit breaker open fallback
func GetCircuitOpenResponse(language string) Response {
	if response, ok := circuitOpenFallbacks[language]; ok {
		return response
	}
	return circuitOpenFallbacks["en"]
}

// IsEmergencyIntent checks if intent requires emergency handling
func IsEmergencyIntent(intent classifier.Intent) bool {
	return intent == classifier.IntentSymptom
}
