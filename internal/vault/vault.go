// Package vault reads and writes Ansible Vault data (format 1.1 and 1.2,
// AES256): whole encrypted files and !vault values in YAML.
package vault

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const header = "$ANSIBLE_VAULT"

// ErrNoPassword: the data is encrypted and no password opens it
var ErrNoPassword = errors.New("vault: no password opens this data (give --vault-password-file or --ask-vault-pass)")

// Secret is one vault password; Label is the vault id (default when empty)
type Secret struct {
	Label    string
	Password []byte
}

// IsEncrypted tells whether data is Ansible Vault data
func IsEncrypted(data []byte) bool {
	return bytes.HasPrefix(bytes.TrimLeft(data, " \t\r\n"), []byte(header+";"))
}

// Decrypt opens vault data with the first secret that fits: the one whose
// label matches the data's vault id first, then the others
func Decrypt(data []byte, secrets []Secret) ([]byte, error) {
	text := strings.TrimSpace(string(data))
	first, body, _ := strings.Cut(text, "\n")
	fields := strings.Split(strings.TrimSpace(first), ";")
	if len(fields) < 3 || fields[0] != header {
		return nil, errors.New("vault: not vault data")
	}
	if fields[1] != "1.1" && fields[1] != "1.2" {
		return nil, fmt.Errorf("vault: format %s is not supported", fields[1])
	}
	if fields[2] != "AES256" {
		return nil, fmt.Errorf("vault: cipher %s is not supported", fields[2])
	}
	label := ""
	if len(fields) > 3 {
		label = fields[3]
	}
	raw, err := hex.DecodeString(strings.Join(strings.Fields(body), ""))
	if err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	parts := bytes.Split(raw, []byte("\n"))
	if len(parts) != 3 {
		return nil, errors.New("vault: malformed data")
	}
	salt, err1 := hex.DecodeString(string(parts[0]))
	mac, err2 := hex.DecodeString(string(parts[1]))
	ciphertext, err3 := hex.DecodeString(string(parts[2]))
	if err := errors.Join(err1, err2, err3); err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	if len(secrets) == 0 {
		return nil, ErrNoPassword
	}
	ordered := make([]Secret, 0, len(secrets))
	for _, s := range secrets {
		if label != "" && s.Label == label {
			ordered = append([]Secret{s}, ordered...)
		} else {
			ordered = append(ordered, s)
		}
	}
	for _, s := range ordered {
		aesKey, macKey, iv, err := deriveKeys(s.Password, salt)
		if err != nil {
			return nil, err
		}
		h := hmac.New(sha256.New, macKey)
		h.Write(ciphertext)
		if !hmac.Equal(h.Sum(nil), mac) {
			continue
		}
		block, err := aes.NewCipher(aesKey)
		if err != nil {
			return nil, err
		}
		plain := make([]byte, len(ciphertext))
		cipher.NewCTR(block, iv).XORKeyStream(plain, ciphertext)
		return unpad(plain)
	}
	return nil, ErrNoPassword
}

// Encrypt writes vault data: format 1.1, or 1.2 with the secret's label
func Encrypt(plain []byte, secret Secret) ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	aesKey, macKey, iv, err := deriveKeys(secret.Password, salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	padded := pad(plain)
	ciphertext := make([]byte, len(padded))
	cipher.NewCTR(block, iv).XORKeyStream(ciphertext, padded)
	h := hmac.New(sha256.New, macKey)
	h.Write(ciphertext)
	inner := hex.EncodeToString(salt) + "\n" + hex.EncodeToString(h.Sum(nil)) + "\n" + hex.EncodeToString(ciphertext)
	body := hex.EncodeToString([]byte(inner))

	var out strings.Builder
	if secret.Label == "" || secret.Label == "default" {
		out.WriteString(header + ";1.1;AES256\n")
	} else {
		out.WriteString(header + ";1.2;AES256;" + secret.Label + "\n")
	}
	for len(body) > 80 {
		out.WriteString(body[:80] + "\n")
		body = body[80:]
	}
	out.WriteString(body + "\n")
	return []byte(out.String()), nil
}

func deriveKeys(password, salt []byte) (aesKey, macKey, iv []byte, err error) {
	key, err := pbkdf2.Key(sha256.New, string(password), salt, 10000, 80)
	if err != nil {
		return nil, nil, nil, err
	}
	return key[:32], key[32:64], key[64:80], nil
}

func pad(data []byte) []byte {
	n := aes.BlockSize - len(data)%aes.BlockSize
	return append(append([]byte{}, data...), bytes.Repeat([]byte{byte(n)}, n)...)
}

func unpad(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, errors.New("vault: bad padding")
	}
	n := int(data[len(data)-1])
	if n == 0 || n > aes.BlockSize || n > len(data) {
		return nil, errors.New("vault: bad padding")
	}
	for _, b := range data[len(data)-n:] {
		if int(b) != n {
			return nil, errors.New("vault: bad padding")
		}
	}
	return data[:len(data)-n], nil
}
