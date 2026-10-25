// Package store persists mounted accounts with AES-256-GCM encrypted
// credentials in data/accounts.json.
package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ccmail/internal/providers"
)

// Paths holds resolved data directories.
var (
	DataDir   string
	CacheDir  string
	DemoFiles string

	keyFile string
	aead    cipher.AEAD
)

// EnsureDirs creates the data layout.
func EnsureDirs(root string) {
	DataDir = filepath.Join(root, "data")
	CacheDir = filepath.Join(DataDir, "cache")
	DemoFiles = filepath.Join(DataDir, "demo-files")
	keyFile = filepath.Join(DataDir, "secret.key")
	for _, d := range []string{DataDir, CacheDir, DemoFiles} {
		_ = os.MkdirAll(d, 0o700)
	}
}

func loadKey() cipher.AEAD {
	if aead != nil {
		return aead
	}
	k, err := os.ReadFile(keyFile)
	if err != nil || len(k) != 32 {
		k = make([]byte, 32)
		_, _ = rand.Read(k)
		_ = os.WriteFile(keyFile, k, 0o600)
	}
	block, _ := aes.NewCipher(k)
	gcm, _ := cipher.NewGCM(block)
	aead = gcm
	return aead
}

// Encrypt seals plaintext with the local key. The wire format is
// std-base64 "iv.tag.ciphertext".
func Encrypt(plain string) string {
	g := loadKey()
	nonce := make([]byte, g.NonceSize())
	_, _ = io.ReadFull(rand.Reader, nonce)
	sealed := g.Seal(nil, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(nonce) + "." +
		base64.StdEncoding.EncodeToString(sealed[:len(sealed)-16]) + "." +
		base64.StdEncoding.EncodeToString(sealed[len(sealed)-16:])
}

func decodeB64(s string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}

// Decrypt opens a payload produced by Encrypt.
func Decrypt(payload string) (string, error) {
	parts := strings.Split(payload, ".")
	if len(parts) != 3 {
		return "", errors.New("bad ciphertext format")
	}
	nonce, err1 := decodeB64(parts[0])
	body, err2 := decodeB64(parts[1])
	tag, err3 := decodeB64(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return "", errors.New("bad ciphertext encoding")
	}
	g := loadKey()
	opened, err := g.Open(nil, nonce, append(body, tag...), nil)
	if err != nil {
		return "", err
	}
	return string(opened), nil
}

// Account is one mounted mailbox (real or demo).
type Account struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"` // "real" | "demo"
	CreatedAt int64             `json:"createdAt"`
	Name      string            `json:"name"`
	Email     string            `json:"email"`
	Color     string            `json:"color"`
	Provider  string            `json:"provider"`
	IMAP      *providers.Server `json:"imap"`
	SMTP      *providers.Server `json:"smtp"`
	Auth      struct {
		User    string `json:"user"`
		PassEnc string `json:"passEnc,omitempty"`
	} `json:"auth"`
}

// Password decrypts the stored credential.
func (a *Account) Password() string {
	if a.Type == "demo" || a.Auth.PassEnc == "" {
		return ""
	}
	pw, err := Decrypt(a.Auth.PassEnc)
	if err != nil {
		return ""
	}
	return pw
}

// Public is the browser-visible shape (never includes credentials).
func (a *Account) Public(extra map[string]any) map[string]any {
	m := map[string]any{
		"id": a.ID, "type": a.Type, "name": a.Name, "email": a.Email,
		"color": a.Color, "provider": a.Provider, "createdAt": a.CreatedAt,
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

var (
	mu       sync.Mutex
	accounts []*Account
)

func file() string { return filepath.Join(DataDir, "accounts.json") }

// Load reads accounts.json (creates nothing on first run).
func Load() []*Account {
	mu.Lock()
	defer mu.Unlock()
	return loadLocked()
}

func loadLocked() []*Account {
	raw, err := os.ReadFile(file())
	if err != nil {
		accounts = nil
		return nil
	}
	var doc struct {
		Accounts []*Account `json:"accounts"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		accounts = nil
		return nil
	}
	// the built-in demo account was removed: drop it if still stored
	accounts = doc.Accounts[:0]
	for _, a := range doc.Accounts {
		if a.Type != "demo" {
			accounts = append(accounts, a)
		}
	}
	if len(accounts) != len(doc.Accounts) {
		saveLocked()
	}
	return accounts
}

func saveLocked() {
	doc := map[string]any{"accounts": accounts}
	raw, _ := json.MarshalIndent(doc, "", "  ")
	tmp := file() + ".tmp"
	_ = os.WriteFile(tmp, raw, 0o600)
	_ = os.Rename(tmp, file())
}

// All returns a copy of the account list.
func All() []*Account {
	mu.Lock()
	defer mu.Unlock()
	out := make([]*Account, len(accounts))
	copy(out, accounts)
	return out
}

// Get finds an account by id.
func Get(id string) *Account {
	mu.Lock()
	defer mu.Unlock()
	for _, a := range accounts {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// AddInput describes a new real account.
type AddInput struct {
	Provider string
	Email    string
	Password string
	Name     string
	Color    string
	IMAP     providers.Server
	SMTP     providers.Server
	AuthUser string
}

// Add stores a new real account.
func Add(in AddInput) *Account {
	id, _ := randomID()
	if in.Name == "" {
		in.Name = strings.SplitN(in.Email, "@", 2)[0]
	}
	a := &Account{
		ID: id, Type: "real", CreatedAt: time.Now().UnixMilli(),
		Name: in.Name, Email: in.Email, Color: in.Color, Provider: in.Provider,
		IMAP: &in.IMAP, SMTP: &in.SMTP,
	}
	a.Auth.User = in.AuthUser
	a.Auth.PassEnc = Encrypt(in.Password)
	mu.Lock()
	accounts = append(accounts, a)
	saveLocked()
	mu.Unlock()
	return a
}

// Patch updates mutable fields.
func Patch(id string, name, color *string) *Account {
	mu.Lock()
	defer mu.Unlock()
	for _, a := range accounts {
		if a.ID != id {
			continue
		}
		if name != nil && *name != "" {
			a.Name = *name
		}
		if color != nil && *color != "" {
			a.Color = *color
		}
		saveLocked()
		return a
	}
	return nil
}

// Remove deletes an account and its cache.
func Remove(id string) bool {
	mu.Lock()
	idx := -1
	for i, a := range accounts {
		if a.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		mu.Unlock()
		return false
	}
	accounts = append(accounts[:idx], accounts[idx+1:]...)
	saveLocked()
	mu.Unlock()
	_ = os.RemoveAll(filepath.Join(CacheDir, SafeDir(id)))
	return true
}

// SafeDir makes an account id safe as a directory name.
func SafeDir(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return fmt.Sprintf("%x", b), nil
}
