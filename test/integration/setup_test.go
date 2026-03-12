//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

func TestMain(m *testing.M) {
	exitCode := 1

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "integration tests skipped: Docker not available (%v). Start Docker and ensure the daemon is reachable (e.g. set DOCKER_HOST if needed).\n", r)
				exitCode = 0
			}
		}()

		ctx := context.Background()

		rabbitmqContainer, err := rabbitmq.Run(ctx, "rabbitmq:3.12-management-alpine")
		if err != nil {
			fmt.Fprintln(os.Stderr, "rabbitmq container failed:", err)
			return
		}
		defer func() { _ = testcontainers.TerminateContainer(rabbitmqContainer) }()

		amqpURL, err := rabbitmqContainer.AmqpURL(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "rabbitmq AmqpURL failed:", err)
			return
		}

		os.Setenv("RABBITMQ_URL", amqpURL)
		exitCode = m.Run()
	}()

	os.Exit(exitCode)
}
