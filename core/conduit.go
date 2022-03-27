package core

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"sync"

	"github.com/TheRebelOfBabylon/Conduit/errors"
	"github.com/TheRebelOfBabylon/Conduit/intercept"
	"github.com/rs/zerolog"
)

const (
	ErrLndNotFound = errors.Error("lnd command not found. Please install lnd to use conduit")
)

var (
	lndLogRegex string = `^[0-9]{4}-[0-9]{2}-[0-9]{2}\s[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]{3}\s\[(?P<LogLvl>INF|TRC|DBG|ERR|WRN|CRT)\]\s(?P<Subsystem>[A-Z]{4}):\s(?P<Text>.+)`
)

// parseLndLog parses the LND log to format it to zerolog
func parseLndLog(scan *bufio.Scanner, log *zerolog.Logger, re *regexp.Regexp, shutdownChan <-chan struct{}, wg sync.WaitGroup) {
	defer wg.Done()
	logger := log.With().Str("process", "LND").Logger()
	for scan.Scan() {
		select {
		case <-shutdownChan:
			return
		default:
			line := scan.Text()
			captures := re.FindStringSubmatch(line)
			logLvl, subName, text := captures[1], captures[2], captures[3]
			switch logLvl {
			case "INF":
				logger.Info().Str("subsystem", subName).Msg(text)
			case "TRC":
				logger.Trace().Str("subsystem", subName).Msg(text)
			case "DBG":
				logger.Debug().Str("subsystem", subName).Msg(text)
			case "ERR":
				logger.Error().Str("subsystem", subName).Msg(text)
			case "WRN":
				logger.Warn().Str("subsystem", subName).Msg(text)
			case "CRT":
				logger.Fatal().Str("subsystem", subName).Msg(text)
			default:
				logger.Error().Msg(fmt.Sprintf("Unkown LND log level: %v", logLvl))
			}
		}
	}
}

// Main is the true entry point for Conduit
func Main(shutdownInterceptor *intercept.Interceptor, cfg *Config, log zerolog.Logger) error {
	var wg sync.WaitGroup
	// Let's check if LND is installed
	if _, err := exec.LookPath("lnd"); err != nil {
		log.Fatal().Msg(ErrLndNotFound.Error())
		return ErrLndNotFound
	}
	// startup LND
	cmd := exec.Command("lnd", "--bitcoin.simnet", "--bitcoin.active", "--bitcoin.node=btcd")
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal().Msg(fmt.Sprint(err))
	}
	scanner := bufio.NewScanner(cmdReader)
	re := regexp.MustCompile(lndLogRegex)
	wg.Add(1)
	go parseLndLog(scanner, &log, re, shutdownInterceptor.ShutdownChannel(), wg)
	if err := cmd.Start(); err != nil {
		log.Fatal().Msg(fmt.Sprint(err))
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal().Msg(fmt.Sprint(err))
	}
	<-shutdownInterceptor.ShutdownChannel()
	wg.Wait()
	return nil
}
