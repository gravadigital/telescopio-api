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
