package auth

import "strings"

type Status struct {
	Available bool
	Source    string
}

func GetAPIKey(env map[string]string) string {
	value := strings.TrimSpace(env["OPENAI_API_KEY"])
	if value == "" {
		return ""
	}

	return value
}

func GetStatus(env map[string]string) Status {
	return Status{
		Available: GetAPIKey(env) != "",
		Source:    "OPENAI_API_KEY",
	}
}
