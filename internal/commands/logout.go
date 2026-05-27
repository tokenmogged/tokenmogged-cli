package commands

import (
	"fmt"

	"github.com/tokenmogged/tokenmogged-cli/internal/config"
)

func Logout() error {
	if err := config.Clear(); err != nil {
		return err
	}
	fmt.Println("Cleared local credentials.")
	return nil
}
