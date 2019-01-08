/*
Copyright © 2018 inwinSTACK.inc

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	goflag "flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/operator"
	"github.com/inwinstack/pa-controller/pkg/pautil"
	"github.com/inwinstack/pa-controller/pkg/version"
	flag "github.com/spf13/pflag"
)

var (
	kubeconfig       string
	host             string
	username         string
	password         string
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

	glog.Infof("Starting PA controller...")

	if ver {
		fmt.Fprintf(os.Stdout, "%s\n", version.GetVersion())
		os.Exit(0)
	}

	if moveType > 6 {
		glog.Fatalf("Error paras: the value must less than or equal to 6.")
	}

	conf := &config.OperatorConfig{
		Kubeconfig:       kubeconfig,
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
