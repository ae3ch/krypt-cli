package crypto

import (
	"strings"
	"testing"
)

// TestEncryptDecryptRoundtrip verifies that Decrypt(Encrypt(…)) returns the original plaintext.
func TestEncryptDecryptRoundtrip(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		title    string
		lang     string
		password string
	}{
		{"plain", "Hello, World!", "", "", ""},
		{"with title", "some content", "My Title", "", ""},
		{"with lang", "package main", "", "go", ""},
		{"with password", "top secret", "Secret", "text", "hunter2"},
		{"multiline", "line1\nline2\nline3", "", "", ""},
		{"unicode", "日本語テスト 🔐", "Unicode Paste", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Encrypt(tc.content, tc.title, tc.lang, tc.password)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}

			if res.EncryptedData == "" || res.IV == "" || res.Salt == "" || res.KeyFragment == "" {
				t.Fatal("one or more fields is empty")
			}

			payload, err := Decrypt(res.EncryptedData, res.IV, res.Salt, res.KeyFragment, tc.password)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}

			if payload.Content != tc.content {
				t.Errorf("content mismatch: got %q, want %q", payload.Content, tc.content)
			}
			if payload.Title != tc.title {
				t.Errorf("title mismatch: got %q, want %q", payload.Title, tc.title)
			}
			if payload.Language != tc.lang {
				t.Errorf("language mismatch: got %q, want %q", payload.Language, tc.lang)
			}
			if payload.KDF != "argon2id" {
				t.Errorf("kdf in payload: got %q, want \"argon2id\"", payload.KDF)
			}
		})
	}
}

// TestWrongPasswordFails ensures decryption fails when the wrong password is used.
func TestWrongPasswordFails(t *testing.T) {
	res, err := Encrypt("secret content", "", "", "correct-password")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = Decrypt(res.EncryptedData, res.IV, res.Salt, res.KeyFragment, "wrong-password")
	if err == nil {
		t.Fatal("expected decryption to fail with wrong password, but it succeeded")
	}
}

// TestNoPasswordRequired verifies a paste without a password decrypts without one.
func TestNoPasswordRequired(t *testing.T) {
	res, err := Encrypt("open content", "", "", "")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	payload, err := Decrypt(res.EncryptedData, res.IV, res.Salt, res.KeyFragment, "")
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if payload.Content != "open content" {
		t.Errorf("unexpected content: %q", payload.Content)
	}
}

// TestCiphertextDoesNotContainPlaintext ensures the server field is not readable.
func TestCiphertextDoesNotContainPlaintext(t *testing.T) {
	secret := "super secret text nobody should see"
	res, err := Encrypt(secret, "My Title", "python", "")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	for _, field := range []string{res.EncryptedData, res.IV, res.Salt} {
		if strings.Contains(field, secret) {
			t.Errorf("server field contains plaintext: %q", field)
		}
		if strings.Contains(field, "My Title") {
			t.Errorf("server field contains title: %q", field)
		}
	}
}

// TestEncryptProducesUniqueValues ensures two calls produce different ciphertexts and keys.
func TestEncryptProducesUniqueValues(t *testing.T) {
	r1, err := Encrypt("same content", "", "", "")
	if err != nil {
		t.Fatalf("Encrypt 1: %v", err)
	}
	r2, err := Encrypt("same content", "", "", "")
	if err != nil {
		t.Fatalf("Encrypt 2: %v", err)
	}

	if r1.KeyFragment == r2.KeyFragment {
		t.Error("two encryptions produced the same key fragment (RNG collision?)")
	}
	if r1.EncryptedData == r2.EncryptedData {
		t.Error("two encryptions produced identical ciphertext")
	}
}
