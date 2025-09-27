package common

import (
	"strings"
	"unicode"
)

// ToSnakeCase converts CamelCase to snake_case
func ToSnakeCase(s string) string {
	var result []byte
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, byte(unicode.ToLower(r)))
	}
	return string(result)
}

// ExtractCommentDescription extracts description from protobuf comments
func ExtractCommentDescription(comments string) string {
	if comments == "" {
		return ""
	}

	// Remove comment markers and extract first meaningful line
	lines := strings.Split(comments, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove common comment markers
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(line)

		if line != "" && !strings.HasPrefix(line, "@") {
			return line
		}
	}
	return ""
}

// SanitizeString removes or replaces characters that might cause issues in generated code
func SanitizeString(s string) string {
	// Replace problematic characters
	s = strings.ReplaceAll(s, "\"", "'")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}

	return strings.TrimSpace(s)
}

// DefaultHTTPMethod returns default HTTP method for gRPC method names
func DefaultHTTPMethod(methodName string) string {
	methodLower := strings.ToLower(methodName)

	switch {
	case strings.HasPrefix(methodLower, "get"), strings.HasPrefix(methodLower, "list"), strings.HasPrefix(methodLower, "find"):
		return "GET"
	case strings.HasPrefix(methodLower, "create"), strings.HasPrefix(methodLower, "add"):
		return "POST"
	case strings.HasPrefix(methodLower, "update"), strings.HasPrefix(methodLower, "modify"), strings.HasPrefix(methodLower, "edit"):
		return "PUT"
	case strings.HasPrefix(methodLower, "delete"), strings.HasPrefix(methodLower, "remove"):
		return "DELETE"
	default:
		return "POST"
	}
}

// GenerateRESTPath generates RESTful path from service and method names
func GenerateRESTPath(serviceName, methodName string) string {
	service := ToSnakeCase(serviceName)
	method := ToSnakeCase(methodName)

	// Remove common suffixes
	service = strings.TrimSuffix(service, "_service")
	service = strings.TrimSuffix(service, "_svc")

	return "/api/v1/" + service + "/" + method
}