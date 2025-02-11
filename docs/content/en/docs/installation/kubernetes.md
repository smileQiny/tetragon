---
title: "Deploy on Kubernetes"
linkTitle: "Kubernetes"
icon: "tutorials"
weight: 1
description: "Deploy and manage Tetragon on Kubernetes"
---

The recommended way to deploy Tetragon on a Kubernetes cluster is to use the
Helm chart with Helm 3. Tetragon uses the [helm.cilium.io](https://helm.cilium.io)
repository to release the helm chart.

## Install

To install the latest release of the Tetragon helm chart, use the following
command.

{{< note >}}
You can find the chart and its documentation with all available values for
configuration in [install/kubernetes](https://github.com/cilium/tetragon/tree/main/install/kubernetes)
in the Tetragon repository. You can use any of the values and override them
with `--set KEY1=VALUE1,KEY2=VALUE2`.
{{< /note >}}

```shell-session
helm repo add cilium https://helm.cilium.io
helm repo update
helm install tetragon cilium/tetragon -n kube-system
```

To wait until Tetragon deployment is ready, use the following `kubectl` command:
```shell-session
kubectl rollout status -n kube-system ds/tetragon -w
```

{{< note >}}
By default, pods in the kube-system namespace are filtered-out.
{{< /note >}}

{{< note >}}
If Tetragon does not to start due to BTF issues, please refer to the
[corresponding question in the FAQ]({{< ref "/docs/faq/#tetragon-failed-to-start-complaining-about-a-missing-btf-file" >}})
for details and solutions.
{{< /note >}}

## Configuration

You can then make modifications to the Tetragon configuration using `helm
upgrade`, see the following example.

```shell-session
helm upgrade tetragon cilium/tetragon -n kube-system --set tetragon.grpc.address=localhost:1337
```

You can also edit the `tetragon-config` ConfigMap directly and restart the
Tetragon daemonset with:

```shell-session
kubectl edit cm tetragon-config -n kube-system
kubectl rollout restart ds/tetragon -n kube-system
```

## Upgrade

Upgrade Tetragon using a new specific version of the helm chart.

```shell-session
helm upgrade tetragon cilium/tetragon -n kube-system --version 0.9.0
```

## Uninstall

Uninstall Tetragon using the following command.

```shell-session
helm uninstall tetragon -n kube-system
```
