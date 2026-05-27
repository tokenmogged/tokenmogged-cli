package hooks

import (
	"net/http"
	"time"
)

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}
