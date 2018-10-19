package main

import (
	goflag "flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/inwinstack/pan-operator/pkg/operator"
	flag "github.com/spf13/pflag"
)

var (
	kubeconfig string
	paHost     string
	paUser     string
	paPassword string
)

func parserFlags() {
	flag.StringVarP(&kubeconfig, "kubeconfig", "", "", "Absolute path to the kubeconfig file.")
	flag.StringVarP(&paHost, "pa-host", "", "", "PAN-OS host address.")
	flag.StringVarP(&paUser, "pa-username", "", "", "PAN-OS username.")
	flag.StringVarP(&paPassword, "pa-password", "", "", "PAN-OS password.")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
}

func main() {
	defer glog.Flush()
	parserFlags()

	glog.Infof("Starting PA operator...")

	f := &operator.Flag{
		Kubeconfig: kubeconfig,
		PAHost:     paHost,
		PAUsername: paUser,
		PAPassword: paPassword,
	}

	op := operator.NewMainOperator(f)
	if err := op.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initing operator instance: %s\n", err)
		os.Exit(1)
	}

	if err := op.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error serving operator instance: %s\n", err)
		os.Exit(1)
	}
}
