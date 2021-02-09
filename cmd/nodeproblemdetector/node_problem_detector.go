/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang/glog"

	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/exporterplugins"
	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/problemdaemonplugins"
	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/exporters"
	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter"
	"k8s.io/node-problem-detector/pkg/exporters/prometheusexporter"
	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/problemdetector"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/version"
)

const (
	serviceControlName = "service-action"
	hourInSeconds      = uint32(60 * 60)
)

func npdInteractive(npdo *options.NodeProblemDetectorOptions) {
	termCh := make(chan error, 1)
	defer close(termCh)

	npdMain(npdo, termCh)
}

func npdMain(npdo *options.NodeProblemDetectorOptions, termCh <-chan error) {

	if npdo.PrintVersion {
		version.PrintVersion()
		return
	}

	npdo.SetNodeNameOrDie()
	npdo.SetConfigFromDeprecatedOptionsOrDie()
	npdo.ValidOrDie()

	// Initialize problem daemons.
	problemDaemons := problemdaemon.NewProblemDaemons(npdo.MonitorConfigPaths)
	if len(problemDaemons) == 0 {
		glog.Fatalf("No problem daemon is configured")
	}

	// Initialize exporters.
	defaultExporters := []types.Exporter{}
	if ke := k8sexporter.NewExporterOrDie(npdo); ke != nil {
		defaultExporters = append(defaultExporters, ke)
		glog.Info("K8s exporter started.")
	}
	if pe := prometheusexporter.NewExporterOrDie(npdo); pe != nil {
		defaultExporters = append(defaultExporters, pe)
		glog.Info("Prometheus exporter started.")
	}

	plugableExporters := exporters.NewExporters()

	npdExporters := []types.Exporter{}
	npdExporters = append(npdExporters, defaultExporters...)
	npdExporters = append(npdExporters, plugableExporters...)

	if len(npdExporters) == 0 {
		glog.Fatalf("No exporter is successfully setup")
	}

	// Initialize NPD core.
	p := problemdetector.NewProblemDetector(problemDaemons, npdExporters)
	if err := p.Run(termCh); err != nil {
		glog.Fatalf("Problem detector failed with error: %v", err)
	}
}

func getProgramPath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}

	if runtime.GOOS == "windows" && filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}

	return "", err
}

func getArgsForService() []string {
	return filterArgsForService(os.Args)
}

func filterArgsForService(args []string) []string {
	newArgs := []string{}
	skip := true

	for _, val := range args {
		isFlagName := strings.HasPrefix(val, "-")
		maybeFlagName := strings.TrimLeft(val, "-")

		if isFlagName {
			if maybeFlagName == serviceControlName {
				skip = true
				continue
			} else if strings.HasPrefix(maybeFlagName, serviceControlName) {
				skip = true
			}
		}

		if skip {
			skip = false
		} else {
			newArgs = append(newArgs, val)
		}
	}

	return newArgs
}
