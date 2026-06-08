// Command scout is the Scout backend HTTP server entrypoint.
package main

import (
	"context"
	"log"

	"scout/internal"
)

func main() {
	ctx := context.Background()
	app, err := internal.NewApp(ctx)
	if err != nil {
		log.Fatalf("failed to initialize scout: %v", err)
	}
	defer app.Close()

	if err := app.Run(ctx); err != nil {
		log.Fatalf("scout exited with error: %v", err)
	}
}
