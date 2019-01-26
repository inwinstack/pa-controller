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
	"log"
	"os"

	"github.com/golang/glog"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/operator"
	"github.com/inwinstack/pa-controller/pkg/version"
	flag "github.com/spf13/pflag"
)

var (
	conf = &config.OperatorConfig{}
	ver  bool
)

func parserFlags() {
	flag.StringVarP(&conf.Kubeconfig, "kubeconfig", "", "", "Absolute path to the kubeconfig file.")
	flag.StringVarP(&conf.Host, "host", "", "", "Palo Alto firewall API host address.")
	flag.StringVarP(&conf.Username, "username", "", "", "Palo Alto firewall API username.")
	flag.StringVarP(&conf.Password, "password", "", "", "Palo Alto firewall API password.")
	flag.StringVarP(&conf.APIKey, "api-key", "", "", "Palo Alto firewall API key.")
	flag.IntVarP(&conf.MoveType, "move-type", "", 5, "The param should be one of the Move constants(0:Skip, 1:Before, 2:DirectlyBefore, 3:After, 4:DirectlyAfter, 5:Top and 6:Bottom).")
	flag.StringVarP(&conf.MoveRule, "move-rule", "", "", "A logical group of security policies somewhere in relation to another security policy.")
	flag.StringVarP(&conf.Vsys, "vsys", "", "", "A virtual system (vsys) is an independent (virtual) firewall instance that you can separately manage within a physical firewall.")
	flag.IntVarP(&conf.Retry, "commit-retry", "", 5, "The number of retry for PA commit job.")
	flag.IntVarP(&conf.CommitWaitTime, "commit-wait-time", "", 2, "The length of time to wait next PA commit.")
	flag.IntVarP(&conf.Interval, "check-failed-interval", "", 30, "The seconds of retry interval for the failed resource.")
	flag.BoolVarP(&ver, "version", "", false, "Display the version.")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
}

func main() {
	defer glog.Flush()
	parserFlags()
	log.SetPrefix("[PA Firewall] ")
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Ltime)

	glog.Infof("Starting PA controller...")

	if ver {
		fmt.Fprintf(os.Stdout, "%s\n", version.GetVersion())
		os.Exit(0)
	}

	if conf.MoveType > 6 {
		glog.Fatalf("Error flag: the value must less than or equal to 6.")
	}

	op := operator.NewMainOperator(conf)
	if err := op.Initialize(); err != nil {
		glog.Fatalf("Error initing operator instance: %v.", err)
	}

	if err := op.Run(); err != nil {
		glog.Fatalf("Error serving operator instance: %s.", err)
	}
}
