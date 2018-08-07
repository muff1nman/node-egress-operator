package stub

import (
	"context"

	ocpv1 "github.com/openshift/api/network/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch event.Object.(type) {
	case *corev1.Node:
		// TODO this
		// algo
		// listOfEgress: pullFromConfig (update this config
		// curOnlineNodes: get online nodes with the correct label
		// curEgNodes: get nodes with egress
		// nodesToClear: curEgNodes - curOnlineNodes
		// curOnlineEgress: getEgress(curOnlineNodes)
		// egressToAdd: listOfEgress - curOnlineEgress
		// egressToRemove: curOnlineEgress - listOfEgress

		listOfEgress := getListOfEgress()

		logrus.Info("Got egress list : %v", listOfEgress)

		curOnlineNodes, err := getCurrentOnlineNodes()
		if err != nil {
			logrus.Errorf("Failed to get current online nodes : %v", err)
			return err
		}

		for index, node := range curOnlineNodes {
			logrus.WithFields(logrus.Fields{
				"name":  node,
				"index": index,
			}).Info("Got onlines node")
		}

		curEgNodes, err := getCurrentEgressNodes()

		if err != nil {
			logrus.Errorf("Failed to get current egress nodes : %v", err)
			return err
		}
	}
	return nil
}

func getListOfEgress() []string {
	rawenv := os.Getenv("EGRESS_LIST")
	return strings.Split(rawenv, ",")
}

func getCurrentOnlineNodes() ([]string, error) {
	listOpts := metav1.ListOptions{}
	nodeList := &corev1.NodeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
	}
	err := sdk.List("", nodeList)
	if err != nil {
		return nil, err
	}
	return FilterAndMapNodes(nodeList.Items, IsNodeOnline), nil
}

func getCurrentEgressNodes() ([]string, error) {
	netList := &ocpv1.HostSubnetList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HostSubnet",
			APIVersion: "v1",
		},
	}
	err := sdk.List("", netList)
	if err != nil {
		return nil, err
	}
	return FilterAndMapNets(netList.Items, HasEgress), nil
}

type nodefilter func(corev1.Node) bool
type netfilter func(ocpv1.HostSubnet) bool

func HasEgress(net ocpv1.HostSubnet) bool {
	return len(net.Spec.EgressIPs) > 0
}

func IsNodeOnline(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func FilterAndMapNodes(vs []corev1.Node, filter nodefilter) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if filter(v) {
			vsf = append(vsf, v.Name)
		}
	}
	return vsf
}

func FilterAndMapNets(vs []ocpv1.HostSubnet, filter netfilter) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if filter(v) {
			vsf = append(vsf, v.Name)
		}
	}
	return vsf
}
