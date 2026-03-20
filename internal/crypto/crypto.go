// Package crypto implements the same AES-256-GCM + Argon2id scheme as the
// krypt browser frontend so the CLI can create and read pastes interoperably.
//
// Encryption flow (mirrors frontend/src/crypto.js):
//
//  1. Generate 32-byte random master key, 12-byte IV, and 16-byte Argon2id salt.
//  2. KDF: Argon2id(masterKey || utf8(password), salt, t=3, m=65536, p=1) → 32-byte AES key.
//     Password is optional; when omitted only masterKey is used as input.
//  3. Encrypt JSON envelope {content, title?, language?, kdf:"argon2id"} with AES-256-GCM.
//  4. All binary values are base64url-encoded (no padding).
//  5. The master key goes in the URL fragment – NEVER sent to the server.
//     Fragment format: "{base64url(masterKey)}"
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	keyLen   = 32 // AES-256
	ivLen    = 12 // 96-bit IV (NIST AES-GCM recommendation)
	saltLen  = 16 // 128-bit Argon2id salt

	// Argon2id parameters — must match frontend/src/crypto.js ARGON2_PARAMS
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 1
)

// Payload is the JSON envelope that gets encrypted.
// Fields match the frontend schema exactly so browser ↔ CLI is interoperable.
type Payload struct {
	Content  string `json:"content"`
	Title    string `json:"title,omitempty"`
	Language string `json:"language,omitempty"`
	KDF      string `json:"kdf,omitempty"`
}

// EncryptResult contains the values that must be sent to the server plus the
// key fragment that must be placed in the URL fragment (never sent to the server).
type EncryptResult struct {
	EncryptedData string // base64url ciphertext (includes GCM auth tag)
	IV            string // base64url 12-byte IV
	Salt          string // base64url 16-byte Argon2id salt
	KeyFragment   string // base64url(masterKey)
}

// Encrypt encrypts content (with optional metadata) using AES-256-GCM + Argon2id.
// password may be empty for no second factor.
func Encrypt(content, title, language, password string) (*EncryptResult, error) {
	masterKey := make([]byte, keyLen)
	if _, err := rand.Read(masterKey); err != nil {
		return nil, fmt.Errorf("rand master key: %w", err)
	}
	iv := make([]byte, ivLen)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("rand iv: %w", err)
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("rand salt: %w", err)
	}

	aesKey := deriveKey(masterKey, salt, password)

	payload := Payload{
		Content:  content,
		Title:    title,
		Language: language,
		KDF:      "argon2id",
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	ciphertext, err := aesGCMEncrypt(aesKey, iv, plaintext)
	if err != nil {
		return nil, err
	}

	return &EncryptResult{
		EncryptedData: toBase64URL(ciphertext),
		IV:            toBase64URL(iv),
		Salt:          toBase64URL(salt),
		KeyFragment:   toBase64URL(masterKey),
	}, nil
}

// Decrypt decrypts a paste given values from the server response and the
// key fragment from the URL (base64url-encoded master key).
// password may be empty.
func Decrypt(encryptedData, ivStr, saltStr, keyFragment, password string) (*Payload, error) {
	ciphertext, err := fromBase64URL(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted_data: %w", err)
	}
	iv, err := fromBase64URL(ivStr)
	if err != nil {
		return nil, fmt.Errorf("decode iv: %w", err)
	}
	salt, err := fromBase64URL(saltStr)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	masterKey, err := fromBase64URL(keyFragment)
	if err != nil {
		return nil, fmt.Errorf("decode key fragment: %w", err)
	}

	aesKey := deriveKey(masterKey, salt, password)

	plaintext, err := aesGCMDecrypt(aesKey, iv, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong key or password?): %w", err)
	}

	var p Payload
	if jsonErr := json.Unmarshal(plaintext, &p); jsonErr != nil || p.Content == "" {
		p = Payload{Content: string(plaintext)}
	}
	return &p, nil
}

// ---- internal helpers -------------------------------------------------------

// deriveKey runs Argon2id with the same parameters as the frontend.
// IKM = masterKey || utf8(password)  (password is optional).
func deriveKey(masterKey, salt []byte, password string) []byte {
	ikm := masterKey
	if password != "" {
		ikm = append(append([]byte(nil), masterKey...), []byte(password)...)
	}
	return argon2.IDKey(ikm, salt, argon2Time, argon2Memory, argon2Threads, keyLen)
}

func aesGCMEncrypt(key, iv, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, ivLen)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, iv, plaintext, nil), nil
}

func aesGCMDecrypt(key, iv, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, ivLen)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, iv, ciphertext, nil)
}

func toBase64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func fromBase64URL(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
