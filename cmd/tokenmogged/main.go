package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tokenmogged/tokenmogged-cli/internal/commands"
)

const version = "0.1.0"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) < 2 {
		commands.Help(version)
		return
	}

	args := os.Args[2:]
	var err error
	switch os.Args[1] {
	case "login":
		err = commands.Login(ctx)
	case "logout":
		err = commands.Logout()
	case "whoami":
		err = commands.Whoami(ctx)
	case "setup":
		err = commands.Setup()
	case "uninstall-hooks":
		err = commands.UninstallHooks()
	case "play":
		err = commands.Play(ctx)
	case "status":
		err = commands.Status(ctx)
	case "submit":
		err = commands.SubmitNow(ctx)
	case "hook":
		err = commands.Hook(args)
	case "version", "--version", "-v":
		fmt.Println("tokenmogged", version)
	case "help", "--help", "-h":
		commands.Help(version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		commands.Help(version)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
