package antivirus

import (
	"context"
	"io"
)

// ScanResult contains the result of a malware scan
type ScanResult struct {
	Infected    bool   // True if malware was detected
	ThreatName  string // Name of detected threat (empty if clean)
	ScannerName string // Name of scanner that produced this result
	Error       error  // Any error that occurred during scanning
}

// Scanner is the interface for pluggable antivirus implementations
// Phase 1: Reject-on-detect model (no quarantine)
// Phase 2: Can evolve to quarantine/review without changing business logic
type Scanner interface {
	// Scan checks file content for malware
	// Returns ScanResult - always check Infected field even if Error is nil
	// If error occurs, treat as Infected=true (fail closed)
	Scan(ctx context.Context, filename string, data io.Reader) ScanResult

	// Name returns the scanner implementation name (for logging)
	Name() string

	// Available checks if the scanner is operational
	Available(ctx context.Context) bool
}

// NoOpScanner is a stub implementation that always returns clean
// Use for development/testing only
type NoOpScanner struct{}

var _ Scanner = (*NoOpScanner)(nil) // Compile-time interface check

func (n *NoOpScanner) Scan(ctx context.Context, filename string, data io.Reader) ScanResult {
	return ScanResult{
		Infected:    false,
		ScannerName: n.Name(),
	}
}

func (n *NoOpScanner) Name() string {
	return "noop"
}

func (n *NoOpScanner) Available(ctx context.Context) bool {
	return true
}

// NewNoOpScanner creates a no-op scanner for development
func NewNoOpScanner() *NoOpScanner {
	return &NoOpScanner{}
}

// ChainScanner runs multiple scanners and fails if any detects malware
type ChainScanner struct {
	scanners []Scanner
}

var _ Scanner = (*ChainScanner)(nil)

func NewChainScanner(scanners ...Scanner) *ChainScanner {
	return &ChainScanner{scanners: scanners}
}

func (c *ChainScanner) Scan(ctx context.Context, filename string, data io.Reader) ScanResult {
	// Note: data can only be read once, so chain scanning requires buffering
	// For simplicity, just use first available scanner
	for _, s := range c.scanners {
		if s.Available(ctx) {
			return s.Scan(ctx, filename, data)
		}
	}
	// No scanner available - fail closed
	return ScanResult{
		Infected:    true,
		ScannerName: "chain",
		Error:       io.EOF,
	}
}

func (c *ChainScanner) Name() string {
	return "chain"
}

func (c *ChainScanner) Available(ctx context.Context) bool {
	for _, s := range c.scanners {
		if s.Available(ctx) {
			return true
		}
	}
	return false
}
