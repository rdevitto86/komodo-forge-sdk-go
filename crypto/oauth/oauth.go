// Package oauth is a deprecated re-export shim for security/oauth.
//
// Deprecated: import github.com/rdevitto86/komodo-forge-sdk-go/security/oauth directly.
package oauth

import o "github.com/rdevitto86/komodo-forge-sdk-go/security/oauth"

func IsValidScope(scope string) bool         { return o.IsValidScope(scope) }
func GetInvalidScopes(scope string) []string { return o.GetInvalidScopes(scope) }
func IsValidGrantType(grantType string) bool { return o.IsValidGrantType(grantType) }
