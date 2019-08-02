/*
Copyright © 2018 inwinSTACK Inc

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
	"context"
	goflag "flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	blendedset "github.com/inwinstack/blended/generated/clientset/versioned"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/ha"
	palog "github.com/inwinstack/pa-controller/pkg/log"
	"github.com/inwinstack/pa-controller/pkg/operator"
	"github.com/inwinstack/pa-controller/pkg/version"
	"github.com/inwinstack/pango"
	flag "github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	cfg             = &config.Config{}
	kubeconfig      string
	haMode          bool
	inspectorSecond int
	ver             bool
)

func parserFlags() {
	flag.StringVarP(&kubeconfig, "kubeconfig", "", "", "Absolute path to the kubeconfig file.")
	flag.IntVarP(&cfg.Threads, "threads", "", 2, "Number of worker threads used by the controller.")
	flag.IntVarP(&cfg.SyncSec, "sync-seconds", "", 60, "Seconds for syncing and retrying objects.")
	flag.StringVarP(&cfg.Host, "host", "", "", "The address of host for the Palo Alto firewall.")
	flag.StringVarP(&cfg.Username, "username", "", "", "The API username of Palo Alto firewall.")
	flag.StringVarP(&cfg.Password, "password", "", "", "The API password of Palo Alto firewall .")
	flag.StringVarP(&cfg.APIKey, "api-key", "", "", "the API key of Palo Alto firewall .")
	flag.IntVarP(&cfg.MoveType, "move-type", "", 5, "The param should be one of the Move constants(0:Skip, 1:Before, 2:DirectlyBefore, 3:After, 4:DirectlyAfter, 5:Top and 6:Bottom).")
	flag.StringVarP(&cfg.MoveRule, "move-rule", "", "", "A logical group of security policies somewhere in relation to another security policy.")
	flag.StringVarP(&cfg.Vsys, "vsys", "", "", "A virtual system (vsys) is an independent (virtual) firewall instance that you can separately manage within a physical firewall.")
	flag.IntVarP(&cfg.Retry, "commit-retry", "", 5, "The number of retry for PA commit job.")
	flag.IntVarP(&cfg.CommitWaitTime, "commit-wait-time", "", 2, "Seconds for waiting next PA commit.")
	flag.StringSliceVarP(&cfg.Admins, "commit-admins", "", []string{"api"}, "Flag commit-admins is an advanced option for doing the partial commit changes by administrators.")
	flag.BoolVarP(&cfg.Force, "force-commit", "", false, "Flag force-commit is if you want to force a commit even if no changes are required.")
	flag.BoolVarP(&cfg.Sync, "sync-commit", "", false, "Flag sync-commit should be true if you want this function to block until the commit job completes.")
	flag.BoolVarP(&cfg.DaNPartial, "dan-partial", "", false, "Flag dan-partial is an advanced option for doing the partial commit for the device and network configuration.")
	flag.BoolVarP(&cfg.PaOPartial, "pao-partial", "", true, "Flag pao-partial is an advanced option for doing the partial commit for the policy and object configuration.")
	flag.BoolVarP(&haMode, "ha", "", false, "Flag ha is an advanced option for enabling high-availability mode.")
	flag.IntVarP(&inspectorSecond, "inspector-seconds", "", 30, "Seconds for checking the high-availability status of PAN.")
	flag.BoolVarP(&ver, "version", "", false, "Display the version.")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
}

func restConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func main() {
	defer glog.Flush()
	log.SetOutput(new(palog.LogWriter))
	parserFlags()

	if ver {
		fmt.Fprintf(os.Stdout, "%s\n", version.GetVersion())
		os.Exit(0)
	}

	fw := &pango.Firewall{Client: pango.Client{
		Hostname: cfg.Host,
		Username: cfg.Username,
		Logging:  pango.LogAction | pango.LogOp,
	}}
	if len(cfg.Password) != 0 {
		fw.Client.Password = cfg.Password
	}

	if len(cfg.APIKey) != 0 {
		fw.Client.ApiKey = cfg.APIKey
	}

	if err := fw.Initialize(); err != nil {
		glog.Fatalf("Error to initialize PAN firewall: %s", err.Error())
	}

	k8scfg, err := restConfig(kubeconfig)
	if err != nil {
		glog.Fatalf("Error to build kubeconfig: %s", err.Error())
	}

	blendedclient, err := blendedset.NewForConfig(k8scfg)
	if err != nil {
		glog.Fatalf("Error to build Blended client: %s", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	op := operator.New(cfg, fw, blendedclient)

	switch haMode {
	case true:
		active := false
		callbacks := &ha.Callbacks{
			OnActive: func() {
				glog.V(3).Infof("PAN firewall on Active.")
				if !active {
					if err := op.Run(ctx); err != nil {
						glog.Fatalf("Error to run the operator: %s.", err)
					}
					active = true
				}
			},
			OnPassive: func() {
				glog.V(3).Infof("PAN firewall on Passive.")
				if active {
					op.Stop()
					active = false
				}
			},
			OnFail: func(err error) {
				op.Stop()
				glog.Errorf("Error to get HA status: %s.", err)
			},
		}

		inspector := ha.NewInspector(fw, inspectorSecond, callbacks)
		if err := inspector.Run(ctx); err != nil {
			glog.Fatalf("Error to run the operator: %s.", err)
		}
	case false:
		if err := op.Run(ctx); err != nil {
			glog.Fatalf("Error to serve the operator instance: %s.", err)
		}
	}

	<-signalChan
	cancel()
	op.Stop()
	glog.Infof("Shutdown signal received, exiting...")
}
