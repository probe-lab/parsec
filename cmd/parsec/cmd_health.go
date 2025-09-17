package main

import (
	"fmt"
	"net/http"

	"github.com/urfave/cli/v2"
)

var HealthCommand = &cli.Command{
	Name: "health",
	Action: func(c *cli.Context) error {
		resp, err := http.Get(fmt.Sprintf("http://%s/readiness", c.Args().First()))
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code: %d", resp.StatusCode)
		}
		fmt.Println("OK")
		return nil
	},
}
