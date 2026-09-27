package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// As in Ansible, a host whose task failed leaves the run and the others go
// on; the play stops when no host is left, with any_errors_fatal, or when
// more than max_fail_percentage of the batch failed.

// hostsFailedError is a task that failed on some of its hosts
type hostsFailedError struct {
	hosts []string
	err   error
}

func (h *hostsFailedError) Error() string { return h.err.Error() }
func (h *hostsFailedError) Unwrap() error { return h.err }

// failures collects the hosts a task failed on
type failures struct {
	hosts []string
	first error
}

func (f *failures) add(host string, err error) {
	f.hosts = append(f.hosts, host)
	if f.first == nil {
		f.first = err
	}
}

func (f *failures) err() error {
	if f.first == nil {
		return nil
	}
	return &hostsFailedError{hosts: f.hosts, err: f.first}
}

// failurePolicy of the play running now
type failurePolicy struct {
	anyErrorsFatal bool
	maxFailPct     int // -1: not set
	batchSize      int
	batchFailed    int
}

type inBlockKey struct{}

// isFailed tells whether a host has left the run
func (e *ExecutionEngine) isFailed(host string) bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.failedHosts[host]
}

// activeHosts drops the hosts that have left the run
func (e *ExecutionEngine) activeHosts(hosts []types.Host) []types.Host {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	if len(e.failedHosts) == 0 {
		return hosts
	}
	active := make([]types.Host, 0, len(hosts))
	for _, h := range hosts {
		if !e.failedHosts[h.Name] {
			active = append(active, h)
		}
	}
	return active
}

// continueAfter decides after a failed task whether the task list goes on
// with the remaining hosts; it returns nil to go on
func (e *ExecutionEngine) continueAfter(ctx context.Context, err error, hosts []types.Host, playResult *types.PlayResult) error {
	var hf *hostsFailedError
	if ctx.Value(inBlockKey{}) != nil || !errors.As(err, &hf) {
		return err
	}
	e.mutex.Lock()
	if e.failedHosts == nil {
		e.failedHosts = map[string]bool{}
	}
	for _, name := range hf.hosts {
		if !e.failedHosts[name] {
			e.failedHosts[name] = true
			e.policy.batchFailed++
		}
	}
	policy := e.policy
	e.mutex.Unlock()
	if playResult != nil {
		playResult.Success = false
	}

	remaining := len(e.activeHosts(hosts))
	switch {
	case policy.anyErrorsFatal:
		return err
	case remaining == 0:
		return err
	case policy.maxFailPct >= 0 && policy.batchSize > 0 &&
		policy.batchFailed*100 > policy.maxFailPct*policy.batchSize:
		return fmt.Errorf("%d of %d hosts failed, more than max_fail_percentage %d%%: %w",
			policy.batchFailed, policy.batchSize, policy.maxFailPct, err)
	}
	e.logger.Warn("Host(s) %s failed and leave the run; %d host(s) go on: %v",
		strings.Join(hf.hosts, ", "), remaining, err)
	return nil
}
