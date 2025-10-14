package ssh

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// keysEqual сравнивает два SSH ключа
func keysEqual(a, b ssh.PublicKey) bool {
	return bytes.Equal(a.Marshal(), b.Marshal())
}

type HostKeyManager struct {
	knownHostsFile string
	knownHosts     map[string]ssh.PublicKey
	mutex          sync.RWMutex
	strictMode     bool
}

func NewHostKeyManager(knownHostsFile string, strictMode bool) *HostKeyManager {
	if knownHostsFile == "" {
		home, _ := os.UserHomeDir()
		knownHostsFile = filepath.Join(home, ".ssh", "known_hosts")
	}

	hkm := &HostKeyManager{
		knownHostsFile: knownHostsFile,
		knownHosts:     make(map[string]ssh.PublicKey),
		strictMode:     strictMode,
	}

	// Load known hosts, ignore errors as file may not exist yet
	_ = hkm.loadKnownHosts()
	return hkm
}

func (hkm *HostKeyManager) loadKnownHosts() error {
	hkm.mutex.Lock()
	defer hkm.mutex.Unlock()

	file, err := os.Open(hkm.knownHostsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Файл не существует, это нормально
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		hosts := strings.Split(parts[0], ",")
		keyType := parts[1]
		keyData := parts[2]

		// Parse the public key using ParseAuthorizedKey
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyType + " " + keyData))
		if err != nil {
			continue
		}

		for _, host := range hosts {
			hkm.knownHosts[host] = key
		}
	}

	return scanner.Err()
}

func (hkm *HostKeyManager) VerifyHostKey(hostname string, remote net.Addr, key ssh.PublicKey) error {
	hkm.mutex.RLock()

	// Проверяем по hostname
	if knownKey, exists := hkm.knownHosts[hostname]; exists {
		hkm.mutex.RUnlock()
		if keysEqual(key, knownKey) {
			return nil
		}
		return fmt.Errorf("host key verification failed for %s: key mismatch", hostname)
	}

	// Проверяем по IP адресу
	if tcpAddr, ok := remote.(*net.TCPAddr); ok {
		ip := tcpAddr.IP.String()
		if knownKey, exists := hkm.knownHosts[ip]; exists {
			hkm.mutex.RUnlock()
			if keysEqual(key, knownKey) {
				return nil
			}
			return fmt.Errorf("host key verification failed for %s (%s): key mismatch", hostname, ip)
		}
	}

	hkm.mutex.RUnlock()

	if hkm.strictMode {
		return fmt.Errorf("host key verification failed for %s: unknown host", hostname)
	}

	// В нестрогом режиме добавляем новый ключ
	return hkm.addHostKey(hostname, key)
}

func (hkm *HostKeyManager) addHostKey(hostname string, key ssh.PublicKey) error {
	hkm.mutex.Lock()
	defer hkm.mutex.Unlock()

	// Добавляем в память
	hkm.knownHosts[hostname] = key

	// Создаем директорию если не существует
	dir := filepath.Dir(hkm.knownHostsFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Добавляем в файл
	file, err := os.OpenFile(hkm.knownHostsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		// Best-effort cleanup in case of panic
		_ = file.Close()
	}()

	keyLine := fmt.Sprintf("%s %s %s\n", hostname, key.Type(),
		strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))

	if _, err = file.WriteString(keyLine); err != nil {
		return err
	}

	// Ensure data is flushed to disk before closing
	if err = file.Sync(); err != nil {
		return err
	}

	// Explicitly close and handle any errors
	return file.Close()
}

func (hkm *HostKeyManager) GetFingerprint(key ssh.PublicKey) string {
	// Use SHA256 instead of MD5 for better security
	hash := sha256.Sum256(key.Marshal())
	// Return base64-encoded SHA256 fingerprint (modern OpenSSH format)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(hash[:])
}

func (hkm *HostKeyManager) IsStrictMode() bool {
	return hkm.strictMode
}

func (hkm *HostKeyManager) SetStrictMode(strict bool) {
	hkm.mutex.Lock()
	defer hkm.mutex.Unlock()
	hkm.strictMode = strict
}
