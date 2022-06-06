package scaler

import (
	"context"
	"fmt"
	"github.com/cobalthq/circleci-k8s-agent/internal/circleci"
	"github.com/cobalthq/circleci-k8s-agent/internal/kubernetes"
	"log"
	"math"
	"strings"
)

type Service struct {
	ci *circleci.Service
	k8s *kubernetes.Service
}

func NewService() (*Service, error) {
	service := &Service{
		ci:  circleci.NewService(),
	}
	k8s, err := kubernetes.NewService()
	if err != nil {
		return nil, err
	}
	service.k8s = k8s
	return service, nil
}

func (s *Service) ScaleRunners(ctx context.Context, namespace string, runnerName string) error {
	activeRunnerCount, err := s.k8s.GetActiveRunnerCount(ctx, namespace, runnerName)
	if err != nil {
		return err
	}

	vars, err := s.k8s.GetRunnerEnvironmentVars(ctx, namespace, runnerName)
	if err != nil {
		return err
	}

	config, err := s.k8s.GetRunnerConfig(ctx, namespace, runnerName)
	if err != nil {
		return err
	}

	token, err := s.k8s.GetCircleToken(ctx, namespace, runnerName)
	if err != nil {
		return err
	}

	waitingJobs, err := s.ci.GetWaitingJobs(token, config.ResourceClass)
	if err != nil {
		return err
	}

	runningJobs, err := s.ci.GetRunningJobs(token, config.ResourceClass)
	if err != nil {
		return err
	}

	pendingPods := int(math.Max(float64(activeRunnerCount - runningJobs), 0))
	jobsToCreate := waitingJobs - pendingPods

	err = s.k8s.SpawnRunners(ctx, namespace, config, vars, jobsToCreate)
	if err != nil {
		return err
	}

	if jobsToCreate > 0 {
		log.Printf("Spawned %d %s/%s runners", jobsToCreate, namespace, runnerName)
	}
	return nil
}

func (s *Service) ScaleAllRunners(ctx context.Context) (error) {
	agentConfig, err := s.k8s.GetAgentConfig(ctx)
	if err != nil {
		return err
	}

	for _, v := range agentConfig.Runners {
		spl := strings.Split(v, "/")
		if len(spl) != 2 {
			return fmt.Errorf("invalid runner specification")
		}
		namespace := spl[0]
		runnerName := spl[1]
		err := s.ScaleRunners(ctx, namespace, runnerName)
		if err != nil {
			return err
		}
	}

	return nil
}
