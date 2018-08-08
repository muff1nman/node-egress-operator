package main

import (
	"context"
	"runtime"

	stub "github.com/muff1nman/node-egress-operator/pkg/stub"
	network "github.com/openshift/api/network/v1"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	printVersion()

	sdk.ExposeMetricsPort()

	k8sutil.AddToSDKScheme(network.AddToScheme)

	resource := "v1"
	kind := "Node"
	resyncPeriod := 5
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, "", resyncPeriod)
	sdk.Watch(resource, kind, "", resyncPeriod)
	sdk.Handle(stub.NewHandler())
	sdk.Run(context.TODO())
}
