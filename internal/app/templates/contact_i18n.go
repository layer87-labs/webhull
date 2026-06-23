package templates

import "github.com/layer87-labs/webhull/internal/pkg/config"

// contactTranslation holds all localised strings and field definitions for the
// built-in contact form in a single language.
type contactTranslation struct {
	heading       string
	submitText    string
	successMsg    string
	successRefMsg string
	errorMsg      string
	fields        []config.FieldConfig
}

// contactTranslations maps language codes to default contact form strings.
// To add a new language, append an entry here — no template logic changes needed.
var contactTranslations = map[string]contactTranslation{
	"ca": {
		heading:       "Contacte",
		submitText:    "Envia el missatge",
		successMsg:    "Moltes gràcies! El teu missatge ha estat enviat.",
		successRefMsg: "La teva referència:",
		errorMsg:      "S'ha produït un error. Si us plau, torna-ho a intentar.",
		fields: []config.FieldConfig{
			{Name: "name", Label: "El teu nom", Type: "text", Required: true, Placeholder: "Anna García"},
			{Name: "email", Label: "Adreça electrònica", Type: "email", Required: true, Placeholder: "anna@exemple.cat"},
			{Name: "subject", Label: "Assumpte", Type: "text", Required: true, Placeholder: "Breu descripció de la teva consulta"},
			{Name: "message", Label: "El teu missatge", Type: "textarea", Required: true, Placeholder: "Explica'm el que necessites..."},
		},
	},
	"en": {
		heading:       "Contact",
		submitText:    "Send message",
		successMsg:    "Thank you! Your message has been sent.",
		successRefMsg: "Your reference number:",
		errorMsg:      "An error occurred. Please try again.",
		fields: []config.FieldConfig{
			{Name: "name", Label: "Name", Type: "text", Required: true, Placeholder: "Jane Smith"},
			{Name: "email", Label: "Email", Type: "email", Required: true, Placeholder: "jane@example.com"},
			{Name: "subject", Label: "Subject", Type: "text", Required: true, Placeholder: "Brief description of your enquiry"},
			{Name: "message", Label: "Message", Type: "textarea", Required: true, Placeholder: "Tell me what you need..."},
		},
	},
	"es": {
		heading:       "Contacto",
		submitText:    "Enviar mensaje",
		successMsg:    "¡Muchas gracias! Tu mensaje ha sido enviado.",
		successRefMsg: "Tu referencia:",
		errorMsg:      "Se ha producido un error. Por favor, inténtalo de nuevo.",
		fields: []config.FieldConfig{
			{Name: "name", Label: "Tu nombre", Type: "text", Required: true, Placeholder: "Ana García"},
			{Name: "email", Label: "Correo electrónico", Type: "email", Required: true, Placeholder: "ana@example.com"},
			{Name: "subject", Label: "Asunto", Type: "text", Required: true, Placeholder: "Breve descripción de tu consulta"},
			{Name: "message", Label: "Tu mensaje", Type: "textarea", Required: true, Placeholder: "Cuéntame lo que necesitas..."},
		},
	},
	"de": {
		heading:       "Kontakt",
		submitText:    "Nachricht senden",
		successMsg:    "Vielen Dank! Ihre Nachricht wurde gesendet.",
		successRefMsg: "Ihre Referenznummer:",
		errorMsg:      "Ein Fehler ist aufgetreten. Bitte versuchen Sie es erneut.",
		fields: []config.FieldConfig{
			{Name: "name", Label: "Name", Type: "text", Required: true},
			{Name: "email", Label: "E-Mail", Type: "email", Required: true},
			{Name: "subject", Label: "Betreff", Type: "text", Required: true},
			{Name: "message", Label: "Nachricht", Type: "textarea", Required: true},
		},
	},
}
