package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/api"
	"github.com/tokenmogged/tokenmogged-cli/internal/config"
)

func Whoami(ctx context.Context) error {
	creds, err := config.Load()
	if err != nil {
		return err
	}
	client := &api.Client{BaseURL: creds.APIBase, Token: creds.Token}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var me api.MeResponse
	if err := client.GetJSON(cctx, "/api/cli/me", &me); err != nil {
		fmt.Printf("Bound locally to %s (%s)\n", creds.Username, creds.Email)
		fmt.Printf("(could not reach server: %v)\n", err)
		return nil
	}
	if me.User == nil {
		fmt.Println("Token rejected by server. Run: tokenmogged login")
		return nil
	}
	fmt.Printf("Signed in as %s\n", me.User.Username)
	fmt.Printf("  email:  %s\n", me.User.Email)
	fmt.Printf("  rating: %d\n", me.User.Rating)
	if me.ActiveMatch != nil {
		fmt.Printf("\nActive match: %s (%s, %s)\n", me.ActiveMatch.ID, me.ActiveMatch.State, me.ActiveMatch.Mode)
	}
	return nil
}
