package proxy

// Authenticator is reserved for Phase 3 username/password authentication.
// Phase 1 only supports unauthenticated SOCKS5 and HTTP CONNECT.
type Authenticator interface {
	Enabled() bool
	Validate(username, password string) bool
}

// NoopAuthenticator disables authentication.
type NoopAuthenticator struct{}

// Enabled returns false because Phase 1 authentication is disabled.
func (NoopAuthenticator) Enabled() bool {
	return false
}

// Validate always returns true when authentication is disabled.
func (NoopAuthenticator) Validate(_, _ string) bool {
	return true
}
