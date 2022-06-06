package circleci

import (
	"fmt"
	"github.com/go-resty/resty/v2"
)

type Service struct {

}

func NewService() *Service {
	return &Service{}
}

type WaitingResponse struct {
	UnclaimedTaskCount int `json:"unclaimed_task_count"`
}

type RunningResponse struct {
	UnclaimedTaskCount int `json:"running_runner_tasks"`
}

func (c *Service) GetWaitingJobs(token string, resourceClass string) (int, error) {
	client := resty.New()
	resp := &WaitingResponse{}
	r, err := client.R().
		SetResult(resp).
		SetHeader("Circle-Token", token).
		Get(fmt.Sprintf("https://runner.circleci.com/api/v2/tasks?resource-class=%s", resourceClass))
	if err != nil {
		return 0, err
	}
	if r.StatusCode() != 200 {
		return 0, fmt.Errorf("circleci returned %d", r.StatusCode())
	}
	return resp.UnclaimedTaskCount, nil
}

func  (c *Service) GetRunningJobs(token string, resourceClass string) (int, error) {
	client := resty.New()
	resp := &RunningResponse{}
	r, err := client.R().
		SetResult(resp).
		SetHeader("Circle-Token", token).
		Get(fmt.Sprintf("https://runner.circleci.com/api/v2/tasks/running?resource-class=%s", resourceClass))
	if err != nil {
		return 0, err
	}
	if r.StatusCode() != 200 {
		return 0, fmt.Errorf("circleci returned %d", r.StatusCode())
	}
	return resp.UnclaimedTaskCount, nil
}