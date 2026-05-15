package sanitization

import "testing"

func TestSanitization_SqlInjectionPattern_Success(t *testing.T) {
	matches := []string{
		"SELECT * FROM users",
		"UNION SELECT password FROM accounts",
		"'; DROP TABLE users; --",
		"exec(xp_cmdshell)",
		"execute stored_proc",
		"CREATE TABLE evil",
		"ALTER TABLE users",
		"INSERT INTO admin",
		"UPDATE users SET",
		"DELETE FROM logs",
		"<script>alert(1)</script>",
		"javascript:void(0)",
		"onerror=alert(1)",
		"onload=evil()",
	}
	for _, input := range matches {
		if !SqlInjectionPattern.MatchString(input) {
			t.Errorf("SqlInjectionPattern should match %q", input)
		}
	}
}

func TestSanitization_SqlInjectionPattern_Failure(t *testing.T) {
	nonMatches := []string{
		"hello world",
		"user@example.com",
		"valid-input-123",
		"/api/users/42",
		"safe text here",
	}
	for _, input := range nonMatches {
		if SqlInjectionPattern.MatchString(input) {
			t.Errorf("SqlInjectionPattern should not match %q", input)
		}
	}
}

func TestSanitization_XssPattern_Success(t *testing.T) {
	matches := []string{
		"<script>alert(1)</script>",
		"</script>",
		"javascript:void(0)",
		"onerror=evil()",
		"onload=attack()",
		"<iframe src=x>",
		"</iframe>",
		"<object data=x>",
		"</object>",
		"<embed src=x>",
		"</embed>",
	}
	for _, input := range matches {
		if !XssPattern.MatchString(input) {
			t.Errorf("XssPattern should match %q", input)
		}
	}
}

func TestSanitization_XssPattern_Failure(t *testing.T) {
	nonMatches := []string{
		"hello world",
		"safe text",
		"/api/v1/resource",
		"user@example.com",
		"normal content here",
	}
	for _, input := range nonMatches {
		if XssPattern.MatchString(input) {
			t.Errorf("XssPattern should not match %q", input)
		}
	}
}

func TestSanitization_PathTraversalPattern_Success(t *testing.T) {
	matches := []string{
		"../etc/passwd",
		"..\\windows\\system32",
		"/var/www/../../../etc",
		"path\\..\\secret",
		"../../sensitive",
	}
	for _, input := range matches {
		if !PathTraversalPattern.MatchString(input) {
			t.Errorf("PathTraversalPattern should match %q", input)
		}
	}
}

func TestSanitization_PathTraversalPattern_Failure(t *testing.T) {
	nonMatches := []string{
		"/api/users/profile",
		"./relative/path",
		"/var/www/html",
		"normal-path/to/file",
		"C:/Windows/System32",
	}
	for _, input := range nonMatches {
		if PathTraversalPattern.MatchString(input) {
			t.Errorf("PathTraversalPattern should not match %q", input)
		}
	}
}

func TestSanitization_NullBytePattern_Success(t *testing.T) {
	matches := []string{
		"before\x00after",
		"\x00start",
		"end\x00",
		"multi\x00ple\x00nulls",
	}
	for _, input := range matches {
		if !NullBytePattern.MatchString(input) {
			t.Errorf("NullBytePattern should match input containing null byte")
		}
	}
}

func TestSanitization_NullBytePattern_Failure(t *testing.T) {
	nonMatches := []string{
		"hello world",
		"safe text",
		"/api/v1/resource",
		"no-null-bytes-here",
	}
	for _, input := range nonMatches {
		if NullBytePattern.MatchString(input) {
			t.Errorf("NullBytePattern should not match %q", input)
		}
	}
}
