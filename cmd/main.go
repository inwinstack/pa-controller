package main

import (
	goflag "flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/inwinstack/pa-operator/pkg/config"
	"github.com/inwinstack/pa-operator/pkg/operator"
	"github.com/inwinstack/pa-operator/pkg/pautil"
	"github.com/inwinstack/pa-operator/pkg/version"
	flag "github.com/spf13/pflag"
)

var (
	kubeconfig       string
	host             string
	username         string
	password         string
	namespaces       []string
	retry            int
	commitTime       int
	moveType         int
	moveRelationRule string
	ver              bool
)

func parserFlags() {
	flag.StringVarP(&kubeconfig, "kubeconfig", "", "", "Absolute path to the kubeconfig file.")
	flag.StringVarP(&host, "pa-host", "", "", "Palo Alto API host address.")
	flag.StringVarP(&username, "pa-username", "", "", "Palo Alto API username.")
	flag.StringVarP(&password, "pa-password", "", "", "Palo Alto API password.")
	flag.StringSliceVarP(&namespaces, "ignore-namespaces", "", nil, "Set ignore namespaces for Kubernetes service.")
	flag.IntVarP(&retry, "retry", "", 5, "Number of retry for PA failed job.")
	flag.IntVarP(&commitTime, "commit-wait-time", "", 2, "The length of time to wait next PA commit.")
	flag.IntVarP(&moveType, "move-type", "", 5, "The param should be one of the Move constants(0:Skip, 1:Before, 2:DirectlyBefore, 3:After, 4:DirectlyAfter, 5:Top and 6:Bottom).")
	flag.StringVarP(&moveRelationRule, "move-relation-rule", "", "", "A logical group of security policies somewhere in relation to another security policy.")
	flag.BoolVarP(&ver, "version", "", false, "Display the version.")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
}

func main() {
	defer glog.Flush()
	parserFlags()

	glog.Infof("Starting PA operator...")

	if ver {
		fmt.Fprintf(os.Stdout, "%s\n", version.GetVersion())
		os.Exit(0)
	}

	if moveType > 6 {
		glog.Fatalf("Error paras: the value must less than or equal to 6.")
	}

	conf := &config.OperatorConfig{
		Kubeconfig:       kubeconfig,
		IgnoreNamespaces: namespaces,
		Retry:            retry,
		CommitWaitTime:   commitTime,
		MoveType:         moveType,
		MoveRelationRule: moveRelationRule,
		PaloAlto: &pautil.Flag{
			Host:     host,
			Username: username,
			Password: password,
		},
	}

	op := operator.NewMainOperator(conf)
	if err := op.Initialize(); err != nil {
		glog.Fatalf("Error initing operator instance: %v.", err)
	}

	if err := op.Run(); err != nil {
		glog.Fatalf("Error serving operator instance: %s.", err)
	}
}
