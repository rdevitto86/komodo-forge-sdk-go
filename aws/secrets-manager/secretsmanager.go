// Package secretsmanager re-exports aws/secretsmanager at the legacy hyphenated import path.
// Services should migrate to github.com/rdevitto86/komodo-forge-sdk-go/aws/secretsmanager.
package secretsmanager

import sm "github.com/rdevitto86/komodo-forge-sdk-go/aws/secretsmanager"

type Config = sm.Config

func Bootstrap(cfg Config) error                                      { return sm.Bootstrap(cfg) }
func GetSecret(key, prefix string) (string, error)                    { return sm.GetSecret(key, prefix) }
func GetSecrets(keys []string, prefix, batchID string) (map[string]string, error) {
	return sm.GetSecrets(keys, prefix, batchID)
}
