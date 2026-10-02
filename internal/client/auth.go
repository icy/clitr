package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rtfpessoa/clitr/internal/log"
	"go.uber.org/zap"
)

const pinEnvFileName = "pin.env"

// UserAgentHeaderValue is the User-Agent header sent to Trade Republic's API.
const UserAgentHeaderValue = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"

// setStdinReader replaces the CLI input source in tests.
func (c *Client) setStdinReader(reader io.Reader) {
	c.stdinReader = reader
}

func (c *Client) collectPin(phoneNumber string) (string, string, error) {
	return collectPinWithReader(bufio.NewReader(c.stdinReader), phoneNumber, c.dataDir)
}

func collectPinWithReader(reader *bufio.Reader, phoneNumber string, dataDir string) (string, string, error) {
	if phoneNumber == "" {
		fmt.Print("Phone number (international format, e.g., +4912345678): ")
		phoneNoInput, err := reader.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("failed to read phone number: %w", err)
		}
		phoneNumber = strings.TrimSpace(phoneNoInput)
	}

	if pin, ok := readPinFromFile(dataDir); ok {
		log.Debug("read PIN from pin.env file")
		return phoneNumber, pin, nil
	}

	fmt.Print("PIN (4 digits): ")
	pinInput, err := reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("failed to read PIN: %w", err)
	}
	return phoneNumber, strings.TrimSpace(pinInput), nil
}

// readPinFromFile reads a PIN from <dataDir>/pin.env, expecting a line like
// `PIN=1234`. It returns ok=false if the file is missing or has no PIN entry,
// so the caller can fall back to prompting the user via stdin.
func readPinFromFile(dataDir string) (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	expandedDir := dataDir
	if strings.HasPrefix(dataDir, "~/") {
		expandedDir = filepath.Join(home, dataDir[2:])
	}

	data, err := os.ReadFile(filepath.Join(expandedDir, pinEnvFileName))
	if err != nil {
		return "", false
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		key, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(key) != "PIN" {
			continue
		}
		pin := strings.Trim(strings.TrimSpace(value), `"'`)
		if pin != "" {
			return pin, true
		}
	}
	return "", false
}

// resumeWebSession attempts to resume a session using saved cookies.
func (c *Client) resumeWebSession() bool {
	if !c.saveCookies {
		return false
	}

	parsedURL, _ := url.Parse(cookiesHost)
	cookies := c.httpClient.Jar.Cookies(parsedURL)
	if len(cookies) == 0 {
		return false
	}

	_, err := c.Settings()
	if err != nil {
		if errors.Is(err, ErrCookiesReset) {
			return false
		}
		log.Error("Failed to get settings", zap.Error(err))
		return false
	}
	return true
}
