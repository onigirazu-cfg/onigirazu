package modules

import (
	"context"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ModuleExecutor defines the interface for executing commands within modules
// This abstracts SSH/local execution to allow testing with mock executors
type ModuleExecutor interface {
	// Execute runs a command and returns stdout
	Execute(command string, args ...string) (string, error)

	// ExecuteContext runs a command with context
	ExecuteContext(ctx context.Context, command string, args ...string) (string, error)

	// Close closes the executor connection
	Close() error

	// SetBecome enables privilege escalation
	SetBecome(become bool, becomeUser, becomeMethod string)

	// GetHost returns the target host
	GetHost() types.Host
}

// RealModuleExecutor wraps the standard CommandExecutor for use as ModuleExecutor
type RealModuleExecutor struct {
	cmdExecutor *executor.CommandExecutor
	host        types.Host
}

// NewRealModuleExecutor creates a new RealModuleExecutor
func NewRealModuleExecutor(host types.Host) (*RealModuleExecutor, error) {
	cmdExec, err := executor.NewCommandExecutor(host)
	if err != nil {
		return nil, err
	}

	return &RealModuleExecutor{
		cmdExecutor: cmdExec,
		host:        host,
	}, nil
}

// NewRealModuleExecutorWithoutPool creates a new RealModuleExecutor without connection pooling
func NewRealModuleExecutorWithoutPool(host types.Host) (*RealModuleExecutor, error) {
	cmdExec, err := executor.NewCommandExecutorWithoutPool(host)
	if err != nil {
		return nil, err
	}

	return &RealModuleExecutor{
		cmdExecutor: cmdExec,
		host:        host,
	}, nil
}

// Execute runs a command and returns stdout
func (r *RealModuleExecutor) Execute(command string, args ...string) (string, error) {
	return r.cmdExecutor.Execute(command, args...)
}

// ExecuteContext runs a command with context
func (r *RealModuleExecutor) ExecuteContext(ctx context.Context, command string, args ...string) (string, error) {
	return r.cmdExecutor.ExecuteContext(ctx, command, args...)
}

// Close closes the executor connection
func (r *RealModuleExecutor) Close() error {
	return r.cmdExecutor.Close()
}

// SetBecome enables privilege escalation
func (r *RealModuleExecutor) SetBecome(become bool, becomeUser, becomeMethod string) {
	r.cmdExecutor.SetBecome(become, becomeUser, becomeMethod)
}

// GetHost returns the target host
func (r *RealModuleExecutor) GetHost() types.Host {
	return r.host
}
