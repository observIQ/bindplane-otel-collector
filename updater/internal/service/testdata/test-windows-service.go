// Copyright  Bindplane, Inc..
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"log"
	"time"

	"golang.org/x/sys/windows/svc"
)

func main() {
	winSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to determine if we were a windows service")
	}

	if !winSvc {
		log.Fatalf("This program must be run as a windows service")
	}

	err = svc.Run("", &windowsService{})
	if err != nil {
		log.Fatalf("Failed to run service: %s", err)
	}

}

type windowsService struct{}

func (sh *windowsService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		req := <-r
		switch req.Cmd {
		case svc.Interrogate:
			s <- req.CurrentStatus
		case svc.Stop, svc.Shutdown:
			time.Sleep(5 * time.Second)
			return false, 0
		default:
			return false, 1052
		}
	}
}
