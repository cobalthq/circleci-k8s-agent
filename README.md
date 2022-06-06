# circleci-k8s-agent

Creates Kubernetes jobs to process CircleCI self-hosted worker queues.

# Usage
This application is Kubernetes native and uses ConfigMaps and secrets for all of its configuration.

It is recommended to use this with [GKE Autopilot](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-overview) so that you can be billed for exact
amounts of resource usage and not have to worry about scaling your nodes.

| Type      | Name                | Namespace           | Purpose                                           |
|-----------|---------------------|---------------------|---------------------------------------------------|
| ConfigMap | circleci-k8s-agent  | namespace of agent  | list of runners                                   |
| ConfigMap | circleci-RUNNER     | namespace of runner | configuration of runner                           |
| Secret    | circleci-RUNNER     | namespace of runner | circleCI and runner tokens                        |
| Secret    | circleci-RUNNER-env | namespace of runner | additional environment variables to mount to jobs |

## Example

These manifests define a runners and deploys the agent. Note that the runner configmaps and secrets live
in the namespace that the jobs will be created in, not the namespace the agent runs in. You can, however, run it all
in one namespace if you'd like.

```
apiVersion: v1
kind: Namespace
metadata:
  name: circleci
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: circleci-k8s-agent
  namespace: circleci
data:
  runners: runners/ruby-highcpu-2 # comma delimited, namespace/runner
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: circleci-k8s-agent
  namespace: circleci
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: circleci-k8s-agent
rules:
  - apiGroups:
      - ""
    resources:
      - configmaps
      - secrets
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - list
  - apiGroups:
      - batch
    resources:
      - jobs
    verbs:
      - create
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: circleci-k8s-agent
  namespace: circleci
subjects:
  - kind: ServiceAccount
    name: circleci-k8s-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: circleci-k8s-agent
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: circleci-k8s-agent
  namespace: circleci
spec:
  replicas: 1
  selector:
    matchLabels:
      app: circleci-k8s-agent
  template:
    metadata:
      labels:
        app: circleci-k8s-agent
    spec:
      serviceAccountName: circleci-k8s-agent
      containers:
        - name: agent
          image: cobaltlabs/circleci-k8s-agent:0.1.0
---
apiVersion: v1
kind: Namespace
metadata:
  name: runners
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: circleci-k8s-agent
  namespace: runners
subjects:
  - kind: ServiceAccount
    name: circleci-k8s-agent
    namespace: circleci
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: circleci-k8s-agent
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: circleci-ruby-highcpu-2
  namespace: runners
data:
  resourceclass: exampleorg/ruby-highcpu-2
  image: exampleorg/ruby
  cpu: 2000m
  memory: 2G
---
apiVersion: v1
kind: Secret
metadata:
  name: circleci-ruby-highcpu-2
  namespace: runners
data:
  token: c29vcGVyc2Vrcml0
  circle-token: c29vcGVyc2Vrcml0
---
apiVersion: v1
kind: Secret
metadata:
  name: circleci-ruby-highcpu-2-env
  namespace: runners
data:
  SUPER_SECRET_VAR: c29vcGVyc2Vrcml0
```
