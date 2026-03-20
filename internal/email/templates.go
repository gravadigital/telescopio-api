package email

import "fmt"

func stageChangeSubject(eventName, newStage string) string {
	return fmt.Sprintf("[Telescopio] El evento «%s» avanzó a la etapa %s", eventName, stageLabel(newStage))
}

func stageChangeBody(eventName, newStage string) string {
	return fmt.Sprintf(`Hola,

Te informamos que el evento «%s» en el que participas ha avanzado a la etapa: %s.

Ingresá a Telescopio para ver los detalles y próximos pasos.

Saludos,
El equipo de Telescopio
`, eventName, stageLabel(newStage))
}

func cancellationSubject(eventName string) string {
	return fmt.Sprintf("[Telescopio] El evento «%s» fue cancelado", eventName)
}

func cancellationBody(eventName string) string {
	return fmt.Sprintf(`Hola,

Lamentamos informarte que el evento «%s» en el que participabas ha sido cancelado.

Si tenés preguntas, comunicate con el organizador del evento.

Saludos,
El equipo de Telescopio
`, eventName)
}

func estimatedDateChangeSubject(eventName, stage string) string {
	return fmt.Sprintf("[Telescopio] Cambio de fecha estimada en el evento «%s»", eventName)
}

func estimatedDateChangeBody(eventName, stage, newDate string) string {
	return fmt.Sprintf(`Hola,

Te informamos que la fecha estimada de cierre de la etapa %s del evento «%s» fue actualizada.

Nueva fecha estimada: %s

Ingresá a Telescopio para ver los detalles.

Saludos,
El equipo de Telescopio
`, stageLabel(stage), eventName, newDate)
}

func passwordResetSubject() string {
	return "[Telescopio] Recuperación de contraseña"
}

func passwordResetBody(resetURL string) string {
	return fmt.Sprintf(`Hola,

Recibimos una solicitud para restablecer la contraseña de tu cuenta en Telescopio.

Hacé clic en el siguiente enlace para crear una nueva contraseña (válido por 1 hora):

%s

Si no solicitaste este cambio, podés ignorar este mensaje. Tu contraseña actual no será modificada.

Saludos,
El equipo de Telescopio
`, resetURL)
}

func stageLabel(stage string) string {
	labels := map[string]string{
		"creation":      "Creación",
		"participation": "Participación",
		"voting":        "Votación",
		"results":       "Resultados",
	}
	if label, ok := labels[stage]; ok {
		return label
	}
	return stage
}
