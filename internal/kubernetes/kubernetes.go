package kubernetes

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/cobalthq/circleci-k8s-agent/internal/core"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"io/ioutil"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"strings"
)

type Service struct {
	clientset *kubernetes.Clientset
}

func NewService() (*Service, error) {
	service := &Service{}
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	service.clientset = clientset
	return service, nil
}

func (k *Service) GetCurrentNamespace() (string, error){
	b, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (k *Service) GetAgentConfig(ctx context.Context) (*core.AgentConfig, error) {

	ac := &core.AgentConfig{}

	namespace, err := k.GetCurrentNamespace()
	if err != nil {
		return nil, err
	}

	cm, err := k.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, "circleci-k8s-agent", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if val, ok := cm.Data["runners"]; ok {
		ac.Runners = strings.Split(val, ",")
	} else {
		return nil, errors.New("runners not specified in configmap")
	}

	return ac, nil

}

func (k *Service) GetRunnerConfig(ctx context.Context, namespace string, runnerName string) (*core.RunnerConfig, error) {
	rc := &core.RunnerConfig{
		Name: runnerName,
	}

	cm, err := k.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, fmt.Sprintf("circleci-%s", runnerName), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if val, ok := cm.Data["resourceclass"]; ok {
		rc.ResourceClass = val
	} else {
		return nil, errors.New("resourceclass not specified in configmap")
	}
	if val, ok := cm.Data["cpu"]; ok {
		rc.CPU, err = resource.ParseQuantity(val)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("cpu not specified in configmap")
	}
	if val, ok := cm.Data["memory"]; ok {
		rc.Memory, err = resource.ParseQuantity(val)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("memory not specified in configmap")
	}
	if val, ok := cm.Data["image"]; ok {
		rc.Image = val
	} else {
		return nil, errors.New("image not specified in configmap")
	}

	return rc, nil
}

func (k *Service) GetRunnerEnvironmentVars(ctx context.Context, namespace string, runnerName string) ([]string, error) {
	secretName :=  fmt.Sprintf("circleci-%s-env", runnerName)
	cm, err := k.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if err.Error() == fmt.Sprintf("secrets \"%s\" not found", secretName) {
			return make([]string, 0), nil
		}
		return nil, err
	}
	stringList := make([]string, 0)
	for k,_ := range cm.Data {
		stringList = append(stringList, k)
	}
	return stringList, nil
}

func (k *Service) GetCircleToken(ctx context.Context, namespace string, runnerName string) (string, error) {
	secret, err := k.clientset.CoreV1().Secrets(namespace).Get(ctx, fmt.Sprintf("circleci-%s", runnerName), metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if val, ok := secret.Data["circle-token"]; ok {
		return string(val), nil
	} else {
		return "", errors.New("circle-token not specified in secret")
	}
}

func (k *Service) GetActiveRunnerCount(ctx context.Context, namespace string, runnerName string) (int, error) {
	pods, err := k.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("circleci-runner=%s", runnerName),
	})

	if err != nil {
		return 0, err
	}

	count := 0
	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning || p.Status.Phase == corev1.PodPending {
			count++
		}
	}

	return count, nil
}

func (s *Service) SpawnWorkers(ctx context.Context, namespace string, config *core.RunnerConfig, envSecrets []string, count int) error {
	env := []corev1.EnvVar{
		{
			Name: "CIRCLECI_RESOURCE_CLASS",
			Value: config.ResourceClass,
		},
		{
			Name: "LAUNCH_AGENT_RUNNER_MODE",
			Value: "single-task",
		},
		{
			Name: "CIRCLECI_API_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef:     &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("circleci-%s", config.Name),
					},
					Key: "runner-token",
				},
			},
		},
	}

	for _,v := range envSecrets {
		env = append(env, corev1.EnvVar{
			Name: v,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef:     &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("circleci-%s-env", config.Name),
					},
					Key: v,
				},
			},
		})
	}

	for i := 1; i <= count; i++ {
		_, err := s.clientset.BatchV1().Jobs(namespace).Create(ctx, &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName:               fmt.Sprintf("%s-", config.Name),
				Namespace:                  namespace,
			},
			Spec:       batchv1.JobSpec{
				Template:                corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"circleci-runner": config.Name,
						},
					},
					Spec:       corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers:                    []corev1.Container{
							{
								Name: "runner",
								Image: config.Image,
								ImagePullPolicy: "IfNotPresent",
								Env: env,
								Resources: corev1.ResourceRequirements{
									Limits:   corev1.ResourceList{
										"cpu": config.CPU,
										"memory": config.Memory,
									},
									Requests: corev1.ResourceList{
										"cpu": config.CPU,
										"memory": config.Memory,
									},
								},
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}
