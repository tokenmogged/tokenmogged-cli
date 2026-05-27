package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/config"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New() (*Client, error) {
	creds, err := config.Load()
	if err != nil {
		return nil, err
	}
	return &Client{
		BaseURL: creds.APIBase,
		Token:   creds.Token,
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func NewAnonymous(baseURL string) *Client {
	return &Client{BaseURL: baseURL, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any, gzipBody bool) (*http.Response, error) {
	var reader io.Reader
	var headers = http.Header{}
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		if gzipBody {
			var gz bytes.Buffer
			gw := gzip.NewWriter(&gz)
			if _, err := gw.Write(buf); err != nil {
				return nil, err
			}
			if err := gw.Close(); err != nil {
				return nil, err
			}
			reader = &gz
			headers.Set("Content-Encoding", "gzip")
		} else {
			reader = bytes.NewReader(buf)
		}
		headers.Set("Content-Type", "application/json")
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header[k] = v
	}
	req.Header.Set("User-Agent", "tokenmogged-cli/0.1.0")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return c.HTTP.Do(req)
}

func (c *Client) PostJSON(ctx context.Context, path string, body any, out any) error {
	resp, err := c.doRequest(ctx, http.MethodPost, path, body, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func (c *Client) PostGzip(ctx context.Context, path string, body any, out any) error {
	resp, err := c.doRequest(ctx, http.MethodPost, path, body, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func (c *Client) GetJSON(ctx context.Context, path string, out any) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func decode(resp *http.Response, out any) error {
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s: %s", resp.Status, string(body))
	}
	if out == nil {
		return nil
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

type AuthCompleteRequest struct {
	Code  string `json:"code"`
	Label string `json:"label,omitempty"`
}

type AuthCompleteResponse struct {
	Token        string `json:"token"`
	CredentialID string `json:"credential_id"`
	User         struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
}

type MeResponse struct {
	User *struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Rating   int    `json:"rating"`
	} `json:"user"`
	ActiveMatch *struct {
		ID    string `json:"id"`
		State string `json:"state"`
		Mode  string `json:"mode"`
	} `json:"active_match"`
}

type ActiveMatch struct {
	MatchID    string `json:"match_id"`
	MatchToken string `json:"match_token"`
	PlayerSide string `json:"player_side"`
	State      string `json:"state"`
	Mode       string `json:"mode"`
	TopicID    string `json:"topic_id"`
	StartedAt  string `json:"started_at"`
}

type ActiveMatchResponse struct {
	Match *ActiveMatch `json:"match"`
}

type StreamEvent struct {
	MatchID     string         `json:"match_id"`
	MatchToken  string         `json:"match_token"`
	EventID     string         `json:"event_id"`
	SessionUUID string         `json:"session_uuid"`
	EventType   string         `json:"event_type"`
	ClientTs    string         `json:"client_ts"`
	ModelID     string         `json:"model_id,omitempty"`
	Tokens      *TokenCounts   `json:"tokens,omitempty"`
	Payload     map[string]any `json:"payload"`
}

type TokenCounts struct {
	Input         int `json:"input"`
	Output        int `json:"output"`
	CacheRead     int `json:"cache_read"`
	CacheCreation int `json:"cache_creation"`
}

type StreamResponse struct {
	Accepted         bool   `json:"accepted"`
	Reason           string `json:"reason,omitempty"`
	CumulativeTokens int    `json:"cumulative_tokens,omitempty"`
	RemainingTokens  int    `json:"remaining_tokens,omitempty"`
	MatchState       string `json:"match_state,omitempty"`
	Warn             string `json:"warn,omitempty"`
}

type SubmissionInitResponse struct {
	CodeUploadURL       string `json:"code_upload_url"`
	TranscriptUploadURL string `json:"transcript_upload_url"`
	CodeBlobURL         string `json:"code_blob_url"`
	TranscriptBlobURL   string `json:"transcript_blob_url"`
	SubmissionID        string `json:"submission_id"`
}

type SubmissionCompleteRequest struct {
	SubmissionID       string         `json:"submission_id"`
	CodeBlobURL        string         `json:"code_blob_url"`
	TranscriptBlobURL  string         `json:"transcript_blob_url"`
	TotalTokens        int            `json:"total_tokens"`
	TotalInputTokens   int            `json:"total_input_tokens"`
	TotalOutputTokens  int            `json:"total_output_tokens"`
	ModelsUsed         map[string]int `json:"models_used"`
	CodeTree           map[string]string `json:"code_tree"`
	EndReason          string         `json:"end_reason"`
}
