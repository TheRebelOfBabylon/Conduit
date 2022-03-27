package core

import (
	"bufio"
	"fmt"
	"os/exec"

	"github.com/TheRebelOfBabylon/Conduit/intercept"
	ps "github.com/mitchellh/go-ps"
	"github.com/rs/zerolog"
)

// Main is the true entry point for Conduit
func Main(shutdownInterceptor *intercept.Interceptor, cfg *Config, log zerolog.Logger) error {
	// Let's check if LND is running, if not we will spin up an instance of LND
	procs, err := ps.Processes()
	if err != nil {
		return nil
	}
	var lndOn bool
	for _, p := range procs {
		if p.Executable() == "lnd" {
			lndOn = true
		}
	}
	if !lndOn {
		log.Debug().Msg("YEET")
		cmd := exec.Command("lnd", "--bitcoin.simnet", "--bitcoin.active", "--bitcoin.node=btcd")
		cmdReader, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal().Msg(fmt.Sprint(err))
		}
		scanner := bufio.NewScanner(cmdReader)
		go func() {
			for scanner.Scan() {
				log.Debug().Msg(scanner.Text())
			}
		}()
		if err := cmd.Start(); err != nil {
			log.Fatal().Msg(fmt.Sprint(err))
		}
		if err := cmd.Wait(); err != nil {
			log.Fatal().Msg(fmt.Sprint(err))
		}
	}
	<-shutdownInterceptor.ShutdownChannel()
	return nil
}
