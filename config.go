package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// userAgent identifies this starter to the API in both HTTP and WS requests.
const userAgent = "livetennisapi-starter-go/1.0.0"

// loadDotEnv loads KEY=value lines from a .env file into the environment.
//
// Tiny on purpose: the starter depends only on the WebSocket library, nothing
// else. Existing environment variables always win, so an exported key is never
// overwritten. A missing file is not an error.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), "'\"")
		if _, ok := os.LookupEnv(key); !ok {
			os.Setenv(key, value)
		}
	}
}

// debugEnabled reports whether LOG_LEVEL=DEBUG, which surfaces every score frame.
func debugEnabled() bool {
	return strings.EqualFold(os.Getenv("LOG_LEVEL"), "DEBUG")
}

// trimFloat renders a float without trailing zeros: 0.63 -> "0.63".
func trimFloat(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}
