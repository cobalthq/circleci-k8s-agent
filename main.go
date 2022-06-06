package main

import (
	"context"
	"github.com/cobalthq/circleci-k8s-agent/internal/scaler"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"log"
	"time"
)

func main() {
	ctx := context.Background()
	service, err := scaler.NewService()
	if err != nil {
		log.Fatalf("scaler.NewService: %q", err)
	}
	for {
		err := service.ScaleAllRunners(ctx)
		if err != nil {
			log.Fatalf("service.ScaleAllRunners: %q", err)
		}
		time.Sleep(15 * time.Second)
	}

}