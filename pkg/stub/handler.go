package stub

import (
	"context"

	"fmt"
	ocpv1 "github.com/openshift/api/network/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	switch o := event.Object.(type) {
	case *corev1.Node:
		logrus.WithFields(logrus.Fields{
			"nodeEventSource": o.Name,
		}).Debug("=========== Running Iteration ==========")

		nodeSelector, err := getNodeSelector()
		if err != nil {
			logrus.Errorf("Failed to get node selector", err)
			return err
		}

		listOfEgress := getListOfEgress()

		curNodes, err := getCurrentNodes(nodeSelector)
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
		curOnlineEgress := FilterAndMapEgress(curHostSubnets, curOnlineNodeNames)
		egressToAdd := SetDifference(listOfEgress, curOnlineEgress)
		egressToRemove := SetDifference(curOnlineEgress, listOfEgress)
		nodeNamesToClear := SetDifference(curEgressNodeNames, curOnlineNodeNames)

		for _, egress := range listOfEgress {
			logrus.WithFields(logrus.Fields{
				"egress": egress,
			}).Debug("Got desired egress")
		}
		for _, node := range curOnlineNodeNames {
			logrus.WithFields(logrus.Fields{
				"name": node,
			}).Debug("Got online node")
		}
		for _, node := range curEgressNodeNames {
			logrus.WithFields(logrus.Fields{
				"name": node,
			}).Debug("Got current egress node")
		}
		for _, egress := range curOnlineEgress {
			logrus.WithFields(logrus.Fields{
				"egress": egress,
			}).Debug("Got online egress")
		}
		for _, node := range nodeNamesToClear {
			logrus.WithFields(logrus.Fields{
				"name": node,
			}).Info("Need to clear egress from offline node")
		}
		for _, egress := range egressToAdd {
			logrus.WithFields(logrus.Fields{
				"egress": egress,
			}).Info("Need to add egress")
		}
		for _, egress := range egressToRemove {
			logrus.WithFields(logrus.Fields{
				"egress": egress,
			}).Info("Need to remove egress")
		}

		modifiedHostSubnetNames := map[string]bool{}

		for _, hostSubnet := range curHostSubnets {
			if Contains(nodeNamesToClear, hostSubnet.Name) {
				hostSubnet.EgressIPs = make([]string, 0)
				modifiedHostSubnetNames[hostSubnet.Name] = true
			} else {
				newEgressList := make([]string, 0)
				for _, egress := range hostSubnet.EgressIPs {
					if Contains(egressToRemove, egress) {
						modifiedHostSubnetNames[hostSubnet.Name] = true
					} else {
						newEgressList = append(newEgressList, egress)
					}
				}
				if modifiedHostSubnetNames[hostSubnet.Name] {
					hostSubnet.EgressIPs = newEgressList
				}
			}
		}

		for len(egressToAdd) > 0 {
			egressFindingAHome := egressToAdd[len(egressToAdd)-1]
			egressToAdd = egressToAdd[:len(egressToAdd)-1]
			var chosenHostSubnet *ocpv1.HostSubnet = nil
			var chosenHostSubnetNumberOfEgress int = -1
			for _, hostSubnet := range curHostSubnets {
				if Contains(curOnlineNodeNames, hostSubnet.Name) && (chosenHostSubnetNumberOfEgress == -1 || chosenHostSubnetNumberOfEgress > len(hostSubnet.EgressIPs)) {
					chosenHostSubnet = &hostSubnet
					chosenHostSubnetNumberOfEgress = len(hostSubnet.EgressIPs)
				}
			}
			if chosenHostSubnet != nil {
				(*chosenHostSubnet).EgressIPs = append((*chosenHostSubnet).EgressIPs, egressFindingAHome)
				modifiedHostSubnetNames[(*chosenHostSubnet).Name] = true
			} else {
				return fmt.Errorf("node-egress-operator: cannot find a home for egressIP %s", egressFindingAHome)
			}
		}

		for _, hostSubnet := range curHostSubnets {
			_, needsUpdate := modifiedHostSubnetNames[hostSubnet.Name]
			if needsUpdate {
				logrus.WithFields(logrus.Fields{
					"hostSubnet": hostSubnet.Name,
					"egressIps":  strings.Join(hostSubnet.EgressIPs, ","),
				}).Info("Updating hostSubnet")
				err := sdk.Update(hostSubnet)
				if err != nil {
					logrus.Errorf("Failed to update hostsubnet", err)
					return err
				}
			}
		}
	}
	return nil
}

func getNodeSelector() (labels.Set, error) {
	rawenv := os.Getenv("NODE_SELECTOR")
	if len(rawenv) > 0 {
		pieces := strings.Split(rawenv, "=")
		if len(pieces) != 2 {
			return nil, fmt.Errorf("node-egress-operator: NODE_SELECTOR must be of the format key=value")
		}
		return map[string]string{pieces[0]: pieces[1]}, nil
	} else {
		return nil, fmt.Errorf("node-egress-operator: NODE_SELECTOR must be set")
	}
}

func getListOfEgress() []string {
	rawenv := os.Getenv("EGRESS_LIST")
	if len(rawenv) > 0 {
		return strings.Split(rawenv, ",")
	} else {
		return nil
	}
}

func getCurrentNodes(sel labels.Set) ([]corev1.Node, error) {
	opts := &metav1.ListOptions{LabelSelector: labels.SelectorFromSet(sel).String()}
	nodeList := &corev1.NodeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
	}
	err := sdk.List("", nodeList, sdk.WithListOptions(opts))
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

func getCurrentHostSubnets() ([]ocpv1.HostSubnet, error) {
	netList := &ocpv1.HostSubnetList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HostSubnet",
			APIVersion: "network.openshift.io/v1",
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

func FilterAndMapEgress(subs []ocpv1.HostSubnet, filterNodeNames []string) []string {
	m := make([]string, 0)
	for _, sub := range subs {
		if Contains(filterNodeNames, sub.Name) {
			for _, egress := range sub.EgressIPs {
				m = append(m, egress)
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

func SetDifference(a []string, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}
