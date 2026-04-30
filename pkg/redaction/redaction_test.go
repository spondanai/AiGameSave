package redaction

import (
	"strings"
	"testing"
)

func TestMaskSecrets_Anthropic(t *testing.T) {
	key := "sk-ant-api03-" + strings.Repeat("A", 32)
	out := MaskSecrets("key is " + key)
	if strings.Contains(out, key) {
		t.Errorf("Anthropic key not redacted: %q", out)
	}
	if !strings.Contains(out, "[REDACTED_API_KEY]") {
		t.Errorf("expected redaction placeholder, got: %q", out)
	}
}

func TestMaskSecrets_OpenAI(t *testing.T) {
	key := "sk-" + strings.Repeat("B", 32)
	out := MaskSecrets("Authorization: " + key)
	if strings.Contains(out, key) {
		t.Errorf("OpenAI key not redacted: %q", out)
	}
}

func TestMaskSecrets_Slack(t *testing.T) {
	key := "xoxb-123456789012-123456789012-" + strings.Repeat("C", 24)
	out := MaskSecrets(key)
	if strings.Contains(out, key) {
		t.Errorf("Slack token not redacted: %q", out)
	}
}

func TestMaskSecrets_GitHub(t *testing.T) {
	for _, prefix := range []string{"ghp_", "gho_", "ghu_", "ghs_", "ghr_"} {
		key := prefix + strings.Repeat("D", 36)
		out := MaskSecrets(key)
		if strings.Contains(out, key) {
			t.Errorf("GitHub token (%s) not redacted: %q", prefix, out)
		}
	}
}

func TestMaskSecrets_AWSAccessKeyID(t *testing.T) {
	key := "AKIA" + strings.Repeat("E", 16)
	out := MaskSecrets("export AWS_ACCESS_KEY_ID=" + key)
	if strings.Contains(out, key) {
		t.Errorf("AWS access key not redacted: %q", out)
	}
}

func TestMaskSecrets_AWSSecretAccessKey(t *testing.T) {
	secret := strings.Repeat("F", 40)
	out := MaskSecrets("aws_secret_access_key = " + secret)
	if strings.Contains(out, secret) {
		t.Errorf("AWS secret key not redacted: %q", out)
	}
}

func TestMaskSecrets_Google(t *testing.T) {
	key := "AIza" + strings.Repeat("G", 35)
	out := MaskSecrets("GOOGLE_API_KEY=" + key)
	if strings.Contains(out, key) {
		t.Errorf("Google API key not redacted: %q", out)
	}
}

func TestMaskSecrets_BearerToken(t *testing.T) {
	token := strings.Repeat("H", 20)
	out := MaskSecrets("Authorization: Bearer " + token)
	if strings.Contains(out, token) {
		t.Errorf("Bearer token not redacted: %q", out)
	}
}

func TestMaskSecrets_PreservesNormalText(t *testing.T) {
	plain := "fix the null pointer dereference in main.go"
	out := MaskSecrets(plain)
	if out != plain {
		t.Errorf("normal text was altered: got %q", out)
	}
}

func TestMaskSecrets_MultipleSecretsInOneString(t *testing.T) {
	anthropic := "sk-ant-api03-" + strings.Repeat("A", 32)
	aws := "AKIA" + strings.Repeat("E", 16)
	input := "ant=" + anthropic + " aws=" + aws
	out := MaskSecrets(input)
	if strings.Contains(out, anthropic) || strings.Contains(out, aws) {
		t.Errorf("not all secrets redacted: %q", out)
	}
	if strings.Count(out, "[REDACTED_API_KEY]") < 2 {
		t.Errorf("expected 2 redaction markers, got: %q", out)
	}
}
