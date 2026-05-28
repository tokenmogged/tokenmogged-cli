package submit

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/api"
	"github.com/tokenmogged/tokenmogged-cli/internal/state"
	"github.com/tokenmogged/tokenmogged-cli/internal/transcript"
)

const (
	MaxUncompressedBytes = 5 * 1024 * 1024
	MaxFileBytes         = 512 * 1024
)

var skipDirs = map[string]bool{
	"node_modules": true,
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
	".git":         true,
	".next":        true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".turbo":       true,
	".cache":       true,
}

var skipSuffixes = []string{".lock", ".log", ".tmp", ".pyc"}

type TranscriptSummary struct {
	SessionUUID       string
	TotalInput        int
	TotalOutput       int
	TotalCacheRead    int
	TotalCacheCreate  int
	ModelsUsed        map[string]int
	EventCount        int
	HasCompactionLine bool
}

func Run(ctx context.Context, client *api.Client, active *state.ActiveMatch, cwd string, endReason string) error {
	tarball, tree, err := tarballDir(cwd)
	if err != nil {
		return fmt.Errorf("tarball: %w", err)
	}

	transcriptPath, summary, err := readTranscript(active)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[tokenmogged] transcript read failed: %v\n", err)
	}
	var transcript []byte
	if transcriptPath != "" {
		transcript, _ = os.ReadFile(transcriptPath)
	}

	var initResp api.SubmissionInitResponse
	if err := client.PostJSON(ctx, "/api/match/"+active.MatchID+"/submission/init", map[string]any{
		"code_bytes":       len(tarball),
		"transcript_bytes": len(transcript),
	}, &initResp); err != nil {
		return fmt.Errorf("submission init: %w", err)
	}

	if err := uploadBlob(ctx, initResp.CodeUploadURL, "application/x-gtar", tarball); err != nil {
		return fmt.Errorf("upload code: %w", err)
	}
	if len(transcript) > 0 {
		if err := uploadBlob(ctx, initResp.TranscriptUploadURL, "application/x-jsonl", transcript); err != nil {
			return fmt.Errorf("upload transcript: %w", err)
		}
	}

	complete := api.SubmissionCompleteRequest{
		SubmissionID:      initResp.SubmissionID,
		TotalInputTokens:  summary.TotalInput,
		TotalOutputTokens: summary.TotalOutput,
		TotalTokens:       summary.TotalInput + summary.TotalOutput + summary.TotalCacheRead + summary.TotalCacheCreate,
		ModelsUsed:        summary.ModelsUsed,
		CodeTree:          tree,
		EndReason:         endReason,
	}
	return client.PostJSON(ctx, "/api/match/"+active.MatchID+"/submission/complete", complete, nil)
}

func tarballDir(root string) ([]byte, map[string]string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tree := map[string]string{}
	var totalBytes int64

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}
		for _, suf := range skipSuffixes {
			if strings.HasSuffix(name, suf) {
				return nil
			}
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > MaxFileBytes {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if !isUTF8(data) {
			return nil
		}
		totalBytes += int64(len(data))
		if totalBytes > MaxUncompressedBytes {
			return errors.New("submission exceeds 5MB cap")
		}

		hdr := &tar.Header{
			Name: rel,
			Mode: 0o644,
			Size: int64(len(data)),
			ModTime: info.ModTime(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
		tree[rel] = string(data)
		return nil
	})

	if err != nil {
		return nil, nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), tree, nil
}

func isUTF8(b []byte) bool {
	if len(b) > 4096 {
		b = b[:4096]
	}
	for _, by := range b {
		if by == 0 {
			return false
		}
	}
	return true
}

func uploadBlob(ctx context.Context, url string, contentType string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-content-type", contentType)
	req.ContentLength = int64(len(body))
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload %s: %s — %s", url, resp.Status, string(b))
	}
	return nil
}

func readTranscript(active *state.ActiveMatch) (string, TranscriptSummary, error) {
	summary := TranscriptSummary{ModelsUsed: map[string]int{}}
	if active == nil {
		return "", summary, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", summary, err
	}
	latest, err := transcript.FindLatest(cwd)
	if err != nil || latest == "" {
		return "", summary, err
	}
	s, err := transcript.ReadFile(latest)
	if err != nil {
		return "", summary, err
	}
	return latest, TranscriptSummary{
		SessionUUID:       s.SessionUUID,
		TotalInput:        s.TotalInput,
		TotalOutput:       s.TotalOutput,
		TotalCacheRead:    s.TotalCacheRead,
		TotalCacheCreate:  s.TotalCacheCreate,
		ModelsUsed:        s.ModelsUsed,
		EventCount:        s.EventCount,
		HasCompactionLine: s.HasCompactionLine,
	}, nil
}

func SortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
