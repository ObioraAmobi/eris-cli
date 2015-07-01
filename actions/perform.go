package actions

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/eris-ltd/eris-cli/chains"
	def "github.com/eris-ltd/eris-cli/definitions"
	"github.com/eris-ltd/eris-cli/services"

	"github.com/eris-ltd/eris-cli/Godeps/_workspace/src/github.com/spf13/cobra"
)

func Do(cmd *cobra.Command, args []string) {
	action, actionVars, err := LoadActionDefinition(args)
	if err != nil {
		logger.Errorln(err)
		return
	}
	err = DoRaw(action, actionVars, cmd.Flags().Lookup("quiet").Changed)
	if err != nil {
		logger.Errorln(err)
		return
	}
}

func DoRaw(action *def.Action, actionVars []string, noOutput bool) error {
	err := StartServicesAndChains(action)
	if err != nil {
		return err
	}

	err = PerformCommand(action, actionVars, noOutput)
	if err != nil {
		return err
	}
	return nil
}

func StartServicesAndChains(action *def.Action) error {
	// start the services and chains
	wg, ch := new(sync.WaitGroup), make(chan error, 1)

	runningServices := services.ListRunningRaw()
	services.StartGroup(ch, wg, action.Services, runningServices, "service", 1, services.StartServiceRaw) // TODO:CNUM

	runningChains := chains.ListRunningRaw()
	services.StartGroup(ch, wg, action.Chains, runningChains, "chain", 1, chains.StartChainRaw)

	go func() {
		wg.Wait()
		ch <- nil
	}()

	return <-ch
}

func PerformCommand(action *def.Action, actionVars []string, noOutput bool) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	actionVars = append(os.Environ(), actionVars...)

	logger.Infoln("Performing Action: ", action.Name)

	for n, step := range action.Steps {
		cmd := exec.Command("sh", "-c", step)
		cmd.Env = actionVars
		cmd.Dir = dir

		prev, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("error running command (%v): %s", err, prev)
		}

		if noOutput {
			logger.Infoln(strings.TrimSpace(string(prev)))
		} else {
			logger.Println(strings.TrimSpace(string(prev)))
		}

		if n != 0 {
			actionVars = actionVars[:len(actionVars)-1]
		}
		actionVars = append(actionVars, ("prev=" + strings.TrimSpace(string(prev))))
	}

	logger.Infoln("Action performed")
	return nil
}
