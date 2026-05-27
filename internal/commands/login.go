package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/api"
	"github.com/tokenmogged/tokenmogged-cli/internal/config"
)

func Login(ctx context.Context) error {
	base := config.APIBase()
	url := strings.TrimRight(base, "/") + "/settings/cli"
	fmt.Println("To bind this machine to your account:")
	fmt.Printf("  1. Open: %s\n", url)
	fmt.Println("  2. Click 'Generate auth code' and copy the code")
	fmt.Println("  3. Paste it here.")
	openBrowser(url)

	fmt.Print("\nAuth code: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	code := strings.TrimSpace(line)
	if code == "" {
		return errors.New("no code entered")
	}

	hostname, _ := os.Hostname()
	label := "CLI on " + hostname

	client := api.NewAnonymous(base)
	var resp api.AuthCompleteResponse
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := client.PostJSON(cctx, "/api/auth/cli/complete", api.AuthCompleteRequest{Code: code, Label: label}, &resp); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	creds := &config.Credentials{
		APIBase:  base,
		Token:    resp.Token,
		UserID:   resp.User.ID,
		Username: resp.User.Username,
		Email:    resp.User.Email,
	}
	if err := config.Save(creds); err != nil {
		return err
	}
	fmt.Printf("\nLogged in as %s (%s).\n", resp.User.Username, resp.User.Email)
	fmt.Println("Run 'tokenmogged setup' next to install Claude Code hooks.")
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}
	_ = cmd.Start()
}
