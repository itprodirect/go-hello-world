package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var sharedHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// Target defines a single check configuration.
type Target struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	Host    string `json:"host,omitempty"`
	Port    int    `json:"port,omitempty"`
	Type    string `json:"type"` // http, tcp, dns
	Timeout int    `json:"timeout_ms,omitempty"`
}

// Result is the outcome of a single check.
type Result struct {
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	Target  string        `json:"target"`
	Status  string        `json:"status"` // up, down, error
	Latency time.Duration `json:"-"`
	Detail  string        `json:"detail,omitempty"`
	TLS     *TLSInfo      `json:"tls,omitempty"`
}

// TLSInfo contains peer certificate summary data.
type TLSInfo struct {
	Subject  string    `json:"subject"`
	Issuer   string    `json:"issuer"`
	NotAfter time.Time `json:"not_after"`
	DaysLeft int       `json:"days_left"`
}

// Check runs one target check according to its type.
func Check(ctx context.Context, target Target) Result {
	timeout := time.Duration(target.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch strings.ToLower(strings.TrimSpace(target.Type)) {
	case "http":
		return checkHTTP(checkCtx, target)
	case "tcp":
		return checkTCP(checkCtx, target)
	case "dns":
		return checkDNS(checkCtx, target)
	default:
		return Result{
			Name:   target.Name,
			Type:   target.Type,
			Target: target.URL,
			Status: "error",
			Detail: fmt.Sprintf("unknown check type: %q", target.Type),
		}
	}
}

func checkHTTP(ctx context.Context, target Target) Result {
	start := time.Now()
	result := Result{
		Name:   target.Name,
		Type:   "http",
		Target: target.URL,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
	if err != nil {
		result.Status = "error"
		result.Detail = fmt.Sprintf("build request: %v", err)
		result.Latency = time.Since(start)
		return result
	}

	resp, err := sharedHTTPClient.Do(req)
	result.Latency = time.Since(start)
	if err != nil {
		result.Status = "down"
		result.Detail = err.Error()
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "up"
	} else {
		result.Status = "down"
	}
	result.Detail = fmt.Sprintf("HTTP %d", resp.StatusCode)

	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		result.TLS = &TLSInfo{
			Subject:  cert.Subject.CommonName,
			Issuer:   cert.Issuer.CommonName,
			NotAfter: cert.NotAfter,
			DaysLeft: int(time.Until(cert.NotAfter).Hours() / 24),
		}
	}

	return result
}

func checkTCP(ctx context.Context, target Target) Result {
	start := time.Now()
	addr := fmt.Sprintf("%s:%d", target.Host, target.Port)
	result := Result{
		Name:   target.Name,
		Type:   "tcp",
		Target: addr,
	}

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	result.Latency = time.Since(start)
	if err != nil {
		result.Status = "down"
		result.Detail = err.Error()
		return result
	}
	_ = conn.Close()

	result.Status = "up"
	result.Detail = "connection successful"

	if target.Port == 443 || target.Port == 8443 {
		tlsConn, err := (&tls.Dialer{
			NetDialer: &net.Dialer{},
			Config: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // probe tool intentionally accepts unknown certs
			},
		}).DialContext(ctx, "tcp", addr)
		if err == nil {
			defer tlsConn.Close()
			tlsClient, ok := tlsConn.(*tls.Conn)
			if ok {
				state := tlsClient.ConnectionState()
				if len(state.PeerCertificates) > 0 {
					cert := state.PeerCertificates[0]
					result.TLS = &TLSInfo{
						Subject:  cert.Subject.CommonName,
						Issuer:   cert.Issuer.CommonName,
						NotAfter: cert.NotAfter,
						DaysLeft: int(time.Until(cert.NotAfter).Hours() / 24),
					}
				}
			}
		}
	}

	return result
}

func checkDNS(ctx context.Context, target Target) Result {
	start := time.Now()
	result := Result{
		Name:   target.Name,
		Type:   "dns",
		Target: target.Host,
	}

	resolver := &net.Resolver{}
	addrs, err := resolver.LookupHost(ctx, target.Host)
	result.Latency = time.Since(start)
	if err != nil {
		result.Status = "down"
		result.Detail = err.Error()
		return result
	}

	result.Status = "up"
	result.Detail = fmt.Sprintf("resolved to %v", addrs)
	return result
}

// LoadTargets loads targets from a JSON file.
func LoadTargets(path string) ([]Target, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read targets file: %w", err)
	}

	var targets []Target
	if err := json.Unmarshal(data, &targets); err != nil {
		return nil, fmt.Errorf("parse targets file: %w", err)
	}

	return targets, nil
}

// MarshalJSON renders Latency as integer milliseconds under latency_ms.
func (r Result) MarshalJSON() ([]byte, error) {
	type resultJSON struct {
		Name      string   `json:"name"`
		Type      string   `json:"type"`
		Target    string   `json:"target"`
		Status    string   `json:"status"`
		LatencyMS int64    `json:"latency_ms"`
		Detail    string   `json:"detail,omitempty"`
		TLS       *TLSInfo `json:"tls,omitempty"`
	}

	return json.Marshal(resultJSON{
		Name:      r.Name,
		Type:      r.Type,
		Target:    r.Target,
		Status:    r.Status,
		LatencyMS: r.Latency.Milliseconds(),
		Detail:    r.Detail,
		TLS:       r.TLS,
	})
}

func StatusEmoji(status string) string {
	switch status {
	case "up":
		return "[OK]"
	case "down":
		return "[FAIL]"
	default:
		return "[ERR]"
	}
}
