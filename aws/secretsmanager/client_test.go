package secretsmanager

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rdevitto86/komodo-forge-sdk-go/testing/testutil"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeSecretsAPI struct {
	getSecretValueFunc func(ctx context.Context, input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

func (f *fakeSecretsAPI) GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if f.getSecretValueFunc != nil {
		return f.getSecretValueFunc(ctx, input, opts...)
	}
	return nil, errors.New("GetSecretValue not configured on fake")
}

// ── Unit Tests ─────────────────────────────────────────────────────────────────

func TestNew_MissingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for empty region, got nil")
	}
}

func TestGetSecret_EmptyName(t *testing.T) {
	c := newWithAPI(&fakeSecretsAPI{}, "")
	defer c.Close()
	_, err := c.GetSecret(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestGetSecrets_NoKeys(t *testing.T) {
	c := newWithAPI(&fakeSecretsAPI{}, "test/path")
	defer c.Close()
	_, err := c.GetSecrets(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty keys, got nil")
	}
}

func TestGetSecrets_EmptySecretPath(t *testing.T) {
	c := newWithAPI(&fakeSecretsAPI{}, "")
	defer c.Close()
	_, err := c.GetSecrets(context.Background(), []string{"KEY"})
	if err == nil {
		t.Fatal("expected error for empty SecretPath, got nil")
	}
}

// ── Component Tests ─────────────────────────────────────────────────────────

func TestGetSecret_Success(t *testing.T) {
	testutil.Component(t)

	const secretName = "prod/db/password"
	const secretValue = "super-secret"

	fake := &fakeSecretsAPI{
		getSecretValueFunc: func(_ context.Context, in *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			if aws.ToString(in.SecretId) != secretName {
				t.Errorf("GetSecretValue called with %q, want %q", aws.ToString(in.SecretId), secretName)
			}
			return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(secretValue)}, nil
		},
	}
	c := newWithAPI(fake, "")
	defer c.Close()

	got, err := c.GetSecret(context.Background(), secretName)
	if err != nil {
		t.Fatalf("GetSecret returned unexpected error: %v", err)
	}
	if got != secretValue {
		t.Errorf("GetSecret = %q, want %q", got, secretValue)
	}
}

func TestGetSecret_CacheHit(t *testing.T) {
	testutil.Component(t)

	calls := 0
	fake := &fakeSecretsAPI{
		getSecretValueFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			calls++
			return &secretsmanager.GetSecretValueOutput{SecretString: aws.String("val")}, nil
		},
	}
	c := newWithAPI(fake, "")
	defer c.Close()

	for range 3 {
		if _, err := c.GetSecret(context.Background(), "my/secret"); err != nil {
			t.Fatalf("GetSecret returned unexpected error: %v", err)
		}
	}
	if calls != 1 {
		t.Errorf("AWS API called %d times, want 1 (cache should serve subsequent calls)", calls)
	}
}

func TestGetSecret_NilSecretString(t *testing.T) {
	testutil.Component(t)

	fake := &fakeSecretsAPI{
		getSecretValueFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{SecretString: nil}, nil
		},
	}
	c := newWithAPI(fake, "")
	defer c.Close()

	_, err := c.GetSecret(context.Background(), "binary/secret")
	if err == nil {
		t.Fatal("expected error for nil SecretString, got nil")
	}
}

func TestGetSecret_APIError(t *testing.T) {
	testutil.Component(t)

	wantErr := errors.New("access denied")
	fake := &fakeSecretsAPI{
		getSecretValueFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return nil, wantErr
		},
	}
	c := newWithAPI(fake, "")
	defer c.Close()

	_, err := c.GetSecret(context.Background(), "my/secret")
	if !errors.Is(err, wantErr) {
		t.Errorf("GetSecret error = %v, want to wrap %v", err, wantErr)
	}
}

func TestGetSecrets_Success(t *testing.T) {
	testutil.Component(t)

	blob, _ := json.Marshal(map[string]string{"DB_PASS": "secret", "API_KEY": "key123", "EXTRA": "unused"})
	fake := &fakeSecretsAPI{
		getSecretValueFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(string(blob))}, nil
		},
	}
	c := newWithAPI(fake, "svc/prod/all")
	defer c.Close()

	got, err := c.GetSecrets(context.Background(), []string{"DB_PASS", "API_KEY"})
	if err != nil {
		t.Fatalf("GetSecrets returned unexpected error: %v", err)
	}
	if got["DB_PASS"] != "secret" {
		t.Errorf("DB_PASS = %q, want %q", got["DB_PASS"], "secret")
	}
	if got["API_KEY"] != "key123" {
		t.Errorf("API_KEY = %q, want %q", got["API_KEY"], "key123")
	}
	if _, ok := got["EXTRA"]; ok {
		t.Error("GetSecrets returned key not in requested list")
	}
}

func TestGetSecrets_CacheHit(t *testing.T) {
	testutil.Component(t)

	calls := 0
	blob, _ := json.Marshal(map[string]string{"K": "v"})
	fake := &fakeSecretsAPI{
		getSecretValueFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			calls++
			return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(string(blob))}, nil
		},
	}
	c := newWithAPI(fake, "svc/prod/all")
	defer c.Close()

	for range 3 {
		if _, err := c.GetSecrets(context.Background(), []string{"K"}); err != nil {
			t.Fatalf("GetSecrets returned unexpected error: %v", err)
		}
	}
	if calls != 1 {
		t.Errorf("AWS API called %d times, want 1 (parsed cache should serve subsequent calls)", calls)
	}
}

func TestGetSecrets_MissingKeys(t *testing.T) {
	testutil.Component(t)

	blob, _ := json.Marshal(map[string]string{"PRESENT": "yes"})
	fake := &fakeSecretsAPI{
		getSecretValueFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(string(blob))}, nil
		},
	}
	c := newWithAPI(fake, "svc/prod/all")
	defer c.Close()

	got, err := c.GetSecrets(context.Background(), []string{"PRESENT", "ABSENT"})
	if err != nil {
		t.Fatalf("GetSecrets returned unexpected error: %v", err)
	}
	if _, ok := got["PRESENT"]; !ok {
		t.Error("GetSecrets missing expected key PRESENT")
	}
	if _, ok := got["ABSENT"]; ok {
		t.Error("GetSecrets returned absent key ABSENT")
	}
}

func TestGetSecrets_NilSecretString(t *testing.T) {
	testutil.Component(t)

	fake := &fakeSecretsAPI{
		getSecretValueFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			return &secretsmanager.GetSecretValueOutput{SecretString: nil}, nil
		},
	}
	c := newWithAPI(fake, "svc/prod/all")
	defer c.Close()

	_, err := c.GetSecrets(context.Background(), []string{"K"})
	if err == nil {
		t.Fatal("expected error for nil SecretString, got nil")
	}
}

// ── Integration Tests ──────────────────────────────────────────────────────────

func TestNew_ValidRegion(t *testing.T) {
	testutil.Integration(t)
	// LocalStack path: only Endpoint set, no real AWS call.
	c, err := New(context.Background(), Config{
		Region:   "us-east-1",
		Endpoint: "http://localhost:4566",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()
}
