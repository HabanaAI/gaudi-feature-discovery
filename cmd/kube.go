package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func kubeClient() (*kubernetes.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("kubeclient: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("kubeclient: %w", err)
	}

	return clientset, nil
}

func getConfig() (*rest.Config, error) {
	kubeConfig := os.Getenv("KUBECONFIG")

	var config *rest.Config
	var err error

	if kubeConfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func updateNodeLabels(ctx context.Context, client kubernetes.Interface, node string, labels map[string]string) error {
	if client == nil {
		return nil
	}
	// Convert labels to a compatible kube API object, so labels can be removed.
	kubeLabels := make(map[string]any, len(labels))
	for k, v := range labels {
		if v == "" {
			kubeLabels[k] = nil
		} else {
			kubeLabels[k] = v
		}
	}

	var nodeData struct {
		Metadata struct {
			Labels map[string]any `json:"labels,omitempty"`
		} `json:"metadata"`
	}
	nodeData.Metadata.Labels = kubeLabels

	patchInfo, err := json.Marshal(nodeData)
	if err != nil {
		return fmt.Errorf("invalid labels values: %w", err)
	}

	updatedNode, err := client.CoreV1().Nodes().Patch(ctx, node, types.MergePatchType, patchInfo, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("patching node labels: %w", err)
	}

	// Verify labels and their values on the updated node
	for k, v := range labels {
		nv, found := updatedNode.Labels[k]
		if !found {
			slog.Warn("failed updating label", "label", k, "original_value", v)
			continue
		}
		if v != nv {
			slog.Warn("node label value mismatch", "label", k, "expected", v, "got", nv)
		}

	}

	return nil
}
