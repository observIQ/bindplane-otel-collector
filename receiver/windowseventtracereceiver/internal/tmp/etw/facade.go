//go:build windows

package etw

import (
	"context"
	"fmt"
)

// CreateMinimalSession creates a very basic ETW session with hardcoded parameters
func CreateMinimalSession(sessionName string) (*MinimalSession, error) {
	// Create a new session
	session := NewMinimalSession(sessionName)

	// Start the session
	if err := session.Start(); err != nil {
		return nil, fmt.Errorf("failed to start minimal session: %w", err)
	}

	// Enable a basic provider
	if err := session.EnableDummyProvider(); err != nil {
		// Try to stop the session before returning the error
		_ = session.Stop(context.Background())
		return nil, fmt.Errorf("failed to enable provider: %w", err)
	}

	return session, nil
}
