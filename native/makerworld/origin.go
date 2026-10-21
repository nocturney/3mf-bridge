// Package makerworld inspects file provenance and validates MakerWorld/Bambu
// native deep links. It NEVER changes default OS handlers or makes requests.
package makerworld

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Source struct {
	Kind       string `json:"kind"` // makerworld, bambu_cdn, unknown
	Confidence string `json:"confidence"`
	OriginURL  string `json:"originUrl,omitempty"`
	Evidence   string `json:"evidence,omitempty"`
}
type Link struct {
	Scheme    string `json:"scheme"`
	RemoteURL string `json:"remoteUrl"`
	Source    Source `json:"source"`
}

func hostnameIs(host, root string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == root || strings.HasSuffix(host, "."+root)
}
func isMakerWorld(u *url.URL) bool {
	return u != nil && (u.Scheme == "https" || u.Scheme == "http") &&
		u.User == nil && hostnameIs(u.Hostname(), "makerworld.com")
}
func isBambuCDN(u *url.URL) bool {
	return u != nil && (u.Scheme == "https" || u.Scheme == "http") &&
		u.User == nil && hostnameIs(u.Hostname(), "bblmw.com")
}
func cleanURL(u *url.URL) string {
	copy := *u
	copy.RawQuery = ""
	copy.Fragment = ""
	copy.User = nil
	return copy.String()
}
func parseOriginURL(raw string) *url.URL {
	raw = strings.TrimSpace(strings.Trim(raw, "\"'<>"))
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil {
		return nil
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		return nil
	}
	if port := u.Port(); port != "" && port != "443" {
		return nil
	}
	return u
}
func Identify(referrer, host string) Source {
	// Referrer from a MakerWorld page is the strongest available signal.
	if u := parseOriginURL(referrer); isMakerWorld(u) {
		return Source{Kind: "makerworld", Confidence: "high", OriginURL: cleanURL(u), Evidence: "download_referrer"}
	}
	// Direct host can also be MakerWorld. CDNs are not conclusive by
	// themselves: Bambu Studio uses them for other interactions too.
	if u := parseOriginURL(host); isMakerWorld(u) {
		return Source{Kind: "makerworld", Confidence: "high", OriginURL: cleanURL(u), Evidence: "download_host"}
	}
	if u := parseOriginURL(host); isBambuCDN(u) {
		return Source{Kind: "bambu_cdn", Confidence: "medium", OriginURL: cleanURL(u), Evidence: "bambu_download_host"}
	}
	return Source{Kind: "unknown", Confidence: "unknown"}
}
// readLimitedADS bounds the memory cost of even a malformed or malicious
// NTFS Zone.Identifier stream before handing its bytes to the parser.
func readLimitedADS(path string) ([]byte, error) {
	stream, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	return io.ReadAll(io.LimitReader(stream, 32*1024+1))
}

func fromWindowsADS(data []byte) Source {
	if len(data) > 32*1024 {
		return Source{Kind: "unknown", Confidence: "unknown"}
	}
	var referrer, host string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if key, value, ok := strings.Cut(line, "="); ok {
			switch strings.ToLower(strings.TrimSpace(key)) {
			case "referrerurl":
				referrer = value
			case "hosturl":
				host = value
			}
		}
	}
	return Identify(referrer, host)
}

var httpLink = regexp.MustCompile(`https?://[^\s"'<>(),]+[^\s"'<>(),.]`)

// fromMacSpotlight reads the Where From URL strings provided by macOS.
// No network, browser or access to user credentials is needed.
func fromMacSpotlight(raw string) Source {
	var firstCDN Source
	for _, candidate := range httpLink.FindAllString(raw, -1) {
		u := parseOriginURL(candidate)
		if isMakerWorld(u) {
			return Source{Kind: "makerworld", Confidence: "high", OriginURL: cleanURL(u), Evidence: "mac_where_from"}
		}
		if isBambuCDN(u) && firstCDN.Kind == "" {
			firstCDN = Source{Kind: "bambu_cdn", Confidence: "medium", OriginURL: cleanURL(u), Evidence: "mac_where_from"}
		}
	}
	if firstCDN.Kind != "" {
		return firstCDN
	}
	return Source{Kind: "unknown", Confidence: "unknown"}
}

func Detect(path string) Source {
	none := Source{Kind: "unknown", Confidence: "unknown"}
	if !strings.EqualFold(filepath.Ext(path), ".3mf") {
		return none
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return none
	}
	switch runtime.GOOS {
	case "windows":
		// ADS path is read-only. It can be missing on copies/non-NTFS volumes.
		data, err := readLimitedADS(path + ":Zone.Identifier")
		if err != nil {
			return none
		}
		return fromWindowsADS(data)
	case "darwin":
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		result, err := exec.CommandContext(ctx, "/usr/bin/mdls", "-raw", "-name", "kMDItemWhereFroms", path).Output()
		if err != nil {
			return none
		}
		return fromMacSpotlight(string(result))
	default:
		return none
	}
}
func unwrapURL(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" || len(input) > 32768 {
		return "", errors.New("invalid external link length")
	}
	for i := 0; i < 2; i++ {
		if strings.HasPrefix(strings.ToLower(input), "https%3a%2f%2f") {
			decoded, err := url.QueryUnescape(input)
			if err != nil {
				return "", fmt.Errorf("invalid encoded link: %w", err)
			}
			input = decoded
		}
	}
	return input, nil
}

// ParseLink handles Bambu Studio's platform-specific URL schemes and our own
// opt-in slicebridge:// link. This does not register or intercept any scheme.
func ParseLink(raw string) (Link, error) {
	result := Link{}
	if len(raw) > 32768 {
		return result, errors.New("external link too long")
	}
	lower := strings.ToLower(raw)
	var encoded string
	switch {
	case strings.HasPrefix(lower, "bambustudio://"):
		outer, err := url.Parse(raw)
		if err != nil || outer.Hostname() != "open" {
			return result, errors.New("unsupported Bambu Studio deep link")
		}
		encoded = outer.Query().Get("file")
		result.Scheme = "bambustudio"
	case strings.HasPrefix(lower, "bambustudioopen://"):
		encoded = raw[len("bambustudioopen://"):]
		result.Scheme = "bambustudioopen"
	case strings.HasPrefix(lower, "slicebridge://"):
		outer, err := url.Parse(raw)
		if err != nil || outer.Hostname() != "import" {
			return result, errors.New("unsupported SliceBridge deep link")
		}
		encoded = outer.Query().Get("url")
		result.Scheme = "slicebridge"
	default:
		return result, errors.New("unsupported external link scheme")
	}
	decoded, err := unwrapURL(encoded)
	if err != nil {
		return Link{}, err
	}
	remote, err := url.Parse(decoded)
	if err != nil || remote.Scheme != "https" || remote.User != nil || remote.Hostname() == "" {
		return Link{}, errors.New("only HTTPS model sources are accepted")
	}
	if remote.Port() != "" && remote.Port() != "443" {
		return Link{}, errors.New("unsupported HTTPS origin port")
	}
	if ip := net.ParseIP(remote.Hostname()); ip != nil {
		return Link{}, errors.New("IP address sources are not accepted")
	}
	result.Source = Identify(decoded, decoded)
	if result.Source.Kind == "unknown" {
		return Link{}, errors.New("external URL is not from MakerWorld or a known Bambu CDN")
	}
	// Preserve the query string here: MakerWorld handoffs use short-lived
	// signed URLs. The URL must not be logged or sent to analytics.
	result.RemoteURL = remote.String()
	return result, nil
}
