package ssh

import (
	"errors"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// a closed port is refused: the dial tries again and gives up after a while
func TestDialRetryingGivesUpOnRefused(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	start := time.Now()
	_, err = dialRetrying(addr, &ssh.ClientConfig{HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: time.Second}) // #nosec G106 -- a refused port
	if err == nil || !isTransientDialError(err) {
		t.Fatalf("expected a refused connection, got %v", err)
	}
	if time.Since(start) < 10*time.Second {
		t.Fatalf("gave up after %s: no retries", time.Since(start))
	}
	if isTransientDialError(errors.New("ssh: handshake failed: no supported methods remain")) {
		t.Fatal("an authentication failure must not be retried")
	}
}
