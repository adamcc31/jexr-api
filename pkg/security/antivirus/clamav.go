package antivirus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// ClamAVScanner connects to clamd daemon for malware scanning
type ClamAVScanner struct {
	address string        // TCP address (host:port) or Unix socket path
	timeout time.Duration // Connection and scan timeout
}

var _ Scanner = (*ClamAVScanner)(nil)

// NewClamAVScanner creates a ClamAV scanner
// address: TCP "localhost:3310" or Unix socket "/var/run/clamav/clamd.sock"
// timeout: Recommended 30-60 seconds for large files
func NewClamAVScanner(address string, timeout time.Duration) *ClamAVScanner {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &ClamAVScanner{
		address: address,
		timeout: timeout,
	}
}

func (c *ClamAVScanner) Name() string {
	return "clamav"
}

// Available checks if ClamAV daemon is reachable
func (c *ClamAVScanner) Available(ctx context.Context) bool {
	network := "tcp"
	if strings.HasPrefix(c.address, "/") {
		network = "unix"
	}

	conn, err := net.DialTimeout(network, c.address, 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Send PING command
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Write([]byte("PING\n"))
	if err != nil {
		return false
	}

	// Read PONG response
	buf := make([]byte, 10)
	n, err := conn.Read(buf)
	if err != nil {
		return false
	}

	return strings.HasPrefix(string(buf[:n]), "PONG")
}

// Scan checks file for malware using ClamAV INSTREAM command
func (c *ClamAVScanner) Scan(ctx context.Context, filename string, data io.Reader) ScanResult {
	result := ScanResult{ScannerName: c.Name()}

	network := "tcp"
	if strings.HasPrefix(c.address, "/") {
		network = "unix"
	}

	// Connect to clamd
	conn, err := net.DialTimeout(network, c.address, c.timeout)
	if err != nil {
		result.Infected = true // Fail closed
		result.Error = fmt.Errorf("failed to connect to clamd: %w", err)
		return result
	}
	defer conn.Close()

	// Set deadline for entire operation
	conn.SetDeadline(time.Now().Add(c.timeout))

	// Send INSTREAM command (null-terminated for zINSTREAM)
	_, err = conn.Write([]byte("zINSTREAM\x00"))
	if err != nil {
		result.Infected = true
		result.Error = fmt.Errorf("failed to send command: %w", err)
		return result
	}

	// Read file data into buffer
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, data)
	if err != nil {
		result.Infected = true
		result.Error = fmt.Errorf("failed to read file data: %w", err)
		return result
	}
	fileData := buf.Bytes()

	// Send file data with size prefix (network byte order, big-endian uint32)
	// Maximum chunk size for ClamAV is usually around 25MB
	chunkSize := uint32(len(fileData))
	sizeBytes := []byte{
		byte(chunkSize >> 24),
		byte(chunkSize >> 16),
		byte(chunkSize >> 8),
		byte(chunkSize),
	}

	_, err = conn.Write(sizeBytes)
	if err != nil {
		result.Infected = true
		result.Error = fmt.Errorf("failed to send size: %w", err)
		return result
	}

	_, err = conn.Write(fileData)
	if err != nil {
		result.Infected = true
		result.Error = fmt.Errorf("failed to send file data: %w", err)
		return result
	}

	// Send end-of-stream marker (4 zero bytes)
	_, err = conn.Write([]byte{0, 0, 0, 0})
	if err != nil {
		result.Infected = true
		result.Error = fmt.Errorf("failed to send end marker: %w", err)
		return result
	}

	// Read response
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil && err != io.EOF {
		result.Infected = true
		result.Error = fmt.Errorf("failed to read response: %w", err)
		return result
	}

	responseStr := strings.TrimSpace(string(response[:n]))

	// Parse response
	// Clean: "stream: OK"
	// Infected: "stream: Eicar-Signature FOUND"
	// Error: "stream: <error message> ERROR"

	if strings.HasSuffix(responseStr, "FOUND") {
		result.Infected = true
		// Extract threat name: "stream: ThreatName FOUND"
		parts := strings.SplitN(responseStr, ":", 2)
		if len(parts) == 2 {
			threatPart := strings.TrimSpace(parts[1])
			threatPart = strings.TrimSuffix(threatPart, " FOUND")
			result.ThreatName = threatPart
		}
	} else if strings.HasSuffix(responseStr, "ERROR") {
		result.Infected = true // Fail closed on scan errors
		result.Error = fmt.Errorf("scan error: %s", responseStr)
	}
	// Otherwise: "OK" means clean

	return result
}
