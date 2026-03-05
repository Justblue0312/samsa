package main

import (
	"context"
	"log"
	"os"

	"github.com/justblue/samsa/bootstrap"
)

// Version is injected using ldflags during build time
var Version = "v0.0.1"

func main() {
	cmd := bootstrap.Serve(Version)

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
