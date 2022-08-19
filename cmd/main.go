package main

import (
	"flag"
	"fmt"
	"os"

	backend "github.com/staticbackendhq/core"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/logger"
)

func main() {
	c := config.LoadConfig()

	log := logger.Get(c)

	var v bool
	flag.BoolVar(&v, "v", false, "Display the version and build info")
	flag.Parse()
	if v {
		fmt.Printf("StaticBackend version %s | %s (%s)\n\n",
			config.Version,
			config.BuildTime,
			config.CommitHash,
		)
		os.Exit(0)
	}

	if len(c.Port) == 0 {
		c.Port = "8099"
	}

	backend.Start(c, log)
}
