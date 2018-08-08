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

		curNodes, err := getCurrentNodes()
		if err != nil {
			logrus.Errorf("Failed to get current nodes : %v", err)
			return err
		}

		curHostSubnets, err := getCurrentHostSubnets()
		if err != nil {
			logrus.Errorf("Failed to get current egress : %v", err)
			return err
		}

		curOnlineNodeNames := FilterAndMapNodes(curNodes, IsNodeOnline)
		curEgressNodeNames := FilterAndMapNets(curHostSubnets, HasEgress)
		curOnlineEgress := FilterAndMapIntoSet(curHostSubnets, curOnlineNodeNames)

		for _, node := range curOnlineNodeNames {
			logrus.WithFields(logrus.Fields{
				"name": node,
			}).Info("Got onlines node")
		}
		for _, node := range curEgressNodeNames {
			logrus.WithFields(logrus.Fields{
				"name": node,
			}).Info("Got current egress node")
		}
		for egress, _ := range curOnlineEgress {
			logrus.WithFields(logrus.Fields{
				"egress": egress,
			}).Info("Got online egress")
		}

		//curEgNodes, err := getCurrentEgressNodes()

		//curOnlineEgress, err := getEgress(curOnlineNodes)

	}
	return nil
}

func getListOfEgress() []string {
	rawenv := os.Getenv("EGRESS_LIST")
	return strings.Split(rawenv, ",")
}

func getCurrentNodes() ([]corev1.Node, error) {
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
	return nodeList.Items, nil
}

func getCurrentHostSubnets() ([]ocpv1.HostSubnet, error) {
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
	return netList.Items, nil
}

type nodefilter func(corev1.Node) bool
type netfilter func(ocpv1.HostSubnet) bool

func HasEgress(net ocpv1.HostSubnet) bool {
	return len(net.EgressIPs) > 0
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

func FilterAndMapIntoSet(subs []ocpv1.HostSubnet, filterNodeNames []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, sub := range subs {
		if Contains(filterNodeNames, sub.Name) {
			for _, egress := range sub.EgressIPs {
				m[egress] = struct{}{}
			}
		}
	}
	return m
}

func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
