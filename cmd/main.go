package main

import (
	backend "github.com/staticbackendhq/core"
	"github.com/staticbackendhq/core/config"
)

func main() {
	c := config.LoadConfig()

	if len(c.Port) == 0 {
		c.Port = "8099"
	}

	backend.Start(c)
}
