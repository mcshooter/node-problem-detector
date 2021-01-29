/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"k8s.io/node-problem-detector/cmd/options"
)

const (
	svcName             = "NodeProblemDetector"
	svcDescription      = "Identifies problems that likely disrupt the operation of Kubernetes workloads."
	svcCommandsAccepted = svc.AcceptStop | svc.AcceptShutdown
)

var (
	elog debug.Log
)

func main() {
	npdo := options.NewNodeProblemDetectorOptions()
	npdo.AddFlags(pflag.CommandLine)

	pflag.Parse()

	runningAsService, err := svc.IsWindowsService()
	if err != nil {
		glog.Errorf("cannot determine if running as Windows Service assuming standalone, %v", err)
		runningAsService = false
	}

	handler := &npdService{
		options: npdo,
	}

	run := debug.Run
	if runningAsService {
		run = svc.Run
	}

	if err := run(svcName, handler); err != nil {
		log.Print(err)
	}
}

type npdService struct {
	sync.Mutex
	options *options.NodeProblemDetectorOptions
}

func (s *npdService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	termCh := make(chan error, 1)
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: svcCommandsAccepted}
	var wg sync.WaitGroup

	options := s.options
	wg.Add(1)

	go func() {
		defer wg.Done()

		npdMain(options, termCh)

		changes <- svc.Status{State: svc.StopPending}
	}()

	serviceLoop(r, changes, termCh)

	wg.Done()

	changes <- svc.Status{State: svc.Stopped, Accepts: svcCommandsAccepted}

	return
}

func serviceLoop(r <-chan svc.ChangeRequest, changes chan<- svc.Status, termCh chan error) {
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				glog.Infof("Stopping %s service, %v", svcName, c.Context)
				termCh <- errors.New("stopping service")
				return
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: svcCommandsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: svcCommandsAccepted}
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
}
