package sanitization

import "regexp"

var (
	SqlInjectionPattern = regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|onerror|onload|<script|</script)`)
	XssPattern = regexp.MustCompile(`(?i)(<script|</script|javascript:|onerror=|onload=|<iframe|</iframe|<object|</object|<embed|</embed)`)
	PathTraversalPattern = regexp.MustCompile(`\.\.\/|\.\.\\`)
	NullBytePattern = regexp.MustCompile(`\x00`)
)
