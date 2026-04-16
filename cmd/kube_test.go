package main

import (
	"context"
	"testing"

	"github.com/habana-internal/habana-feature-discovery/collector"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	corefake "k8s.io/client-go/kubernetes/fake"
)

// Simple sanity test
func TestUpdateNodeLabels(t *testing.T) {
	t.Run("node labels are updated with expected values", nodeLabelsUpdated)
	t.Run("node labels are removed when label value is empty", nodeLabelsRemoved)
}

func nodeLabelsUpdated(t *testing.T) {
	ctx := context.Background()

	emptyNode := corev1.Node{
		ObjectMeta: v1.ObjectMeta{
			Name:   "empty-node",
			Labels: make(map[string]string),
		},
	}
	fclient := corefake.NewClientset(&emptyNode)

	devID := "1020"
	labels := map[string]string{
		collector.DeviceIDLabel: devID,
	}
	err := updateNodeLabels(ctx, fclient, "empty-node", labels)
	if err != nil {
		t.Fatalf("was not expecting error ,got %v", err)
	}

	// Get node and check label
	updatedNode, err := fclient.Tracker().Get(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "nodes",
	}, "", emptyNode.Name)
	if err != nil {
		t.Fatalf("was not expecting error ,got %v", err)
	}

	unode, ok := updatedNode.(*corev1.Node)
	if !ok {
		t.Fatalf("expected node but got %T", updatedNode)
	}

	val, found := unode.Labels[collector.DeviceIDLabel]
	if !found {
		t.Errorf("expected label %q to exist, but it's not", collector.DeviceIDLabel)
	}
	if val != devID {
		t.Errorf("vendor=%q, expected %q", val, devID)
	}
}

func nodeLabelsRemoved(t *testing.T) {
	ctx := context.Background()

	node1 := corev1.Node{
		ObjectMeta: v1.ObjectMeta{
			Name: "node1",
			Labels: map[string]string{
				"habana.ai/driver": "1.17.0",
			},
		},
	}
	fclient := corefake.NewClientset(&node1)

	labels := map[string]string{
		collector.DriverVersionLabel: "",
	}
	err := updateNodeLabels(ctx, fclient, "node1", labels)
	if err != nil {
		t.Fatalf("was not expecting error ,got %v", err)
	}

	// Get node and check label
	updatedNode, err := fclient.Tracker().Get(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "nodes",
	}, "", node1.Name)
	if err != nil {
		t.Fatalf("was not expecting error ,got %v", err)
	}

	unode, ok := updatedNode.(*corev1.Node)
	if !ok {
		t.Fatalf("expected node but got %T", updatedNode)
	}

	_, found := unode.Labels[collector.DeviceIDLabel]
	if found {
		t.Errorf("expected label %q to be removed, but it's not", collector.DriverVersionLabel)
	}
}
