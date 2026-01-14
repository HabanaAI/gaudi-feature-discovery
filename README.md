# HabanaLabs feature discovery

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Table of Contents

- [HabanaLabs feature discovery](#habanalabs-feature-discovery)
  * [Overview](#overview)
  * [Prerequisites](#prerequisites)
  * [Quick Start](#quick-start)
    + [Node Feature Discovery (NFD)](#node-feature-discovery-nfd)
    + [Deploy Habana Feature Discovery (HFD)](#deploy-habana-feature-discovery-hfd)
      - [Deamonset](#deamonset)
    + [Verifying Everything Works](#verifying-everything-works)
  * [The HFD Command line interface](#the-hfd-command-line-interface)


## Overview

Habana Feature Discovery for Kubernetes is a software component that allows
you to automatically generate labels for the set of Habana devices available on a node.
It leverages the [Node Feature Discovery](https://github.com/kubernetes-sigs/node-feature-discovery)
to perform this labeling.

## Prerequisites

The list of prerequisites for running the Habana Feature Discovery is
described below:
* Kubernetes version >= 1.10
* Habana device plugin for Kubernetes (see how to [setup](https://github.com/HabDevops/habanalabs-k8s-device-plugin))
* NFD deployed on each node you want to label with the local source configured
  * To deploy NFD, please see https://github.com/kubernetes-sigs/node-feature-discovery


**Note:** The following assumes you have at least one node in your cluster with Habana device.

### Node Feature Discovery (NFD)

The first step is to make sure that [Node Feature Discovery](https://github.com/kubernetes-sigs/node-feature-discovery)
is running on every node you want to label.

You also need to configure the `Node Feature Discovery` to only expose vendor
IDs in the PCI source. To do so, please refer to the Node Feature Discovery
documentation.

### Deploy Habana Feature Discovery (HFD)

The next step is to run Habana Feature Discovery on each node as a Deamonset
or as a Job.

#### Deamonset

```shell
kubectl apply -f https://raw.githubusercontent.com/HabanaAI/habanalabs-feature-discovery/habanalabs-feature-discovery-daemonset.yaml
```

### Verifying Everything Works

With both NFD and GFD deployed and running, you should now be able to see Habana devices
related labels appearing on any nodes that have Habana devices installed on them.

```
$ kubectl get nodes -o yaml
apiVersion: v1
items:
- apiVersion: v1
  kind: Node
  metadata:
    ...

    labels:
      habana.ai/device.count=8
      habana.ai/hfd.timestamp=1600440374
      habana.ai/machine.type=KVM
      habana.ai/driver.version=0.11.0-26305c5e
      ...
...

```

## The HFD Command line interface

Available options:
```
habanalabs-feature-discovery:
Usage:
  habanalabs-feature-discovery [--once | --interval=<seconds>] [--output-file=<file> | -o <file>]
  habanalabs-feature-discovery -h | --help
  habanalabs-feature-discovery --version

Options:
  -h --help                       Show this help message and exit
  --version                       Display version and exit
  --once                          Label once and exit
  --interval=<seconds>            Time to sleep between labeling [Default: 60s]
  -o <file> --output-file=<file>  Path to output file
                                  [Default: /etc/kubernetes/node-feature-discovery/features.d/hfd]
```

You can also use environment variables:

| Env Variable       | Option           | Example |
| ------------------ | ---------------- | ------- |
| HFD_ONCE           | --once           | TRUE    |
| HFD_OUTPUT_FILE    | --output-file    | output  |
| HFD_INTERVAL       | --interval       | 10s     |
| HFD_ROOT_PREFIX    |                  |         |

Environment variables override the command line options if they conflict.
