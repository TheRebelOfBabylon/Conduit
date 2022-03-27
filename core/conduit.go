/*
Copyright (C) 2015-2018 Lightning Labs and The Lightning Network Developers
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package core

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"sync"

	"github.com/TheRebelOfBabylon/Conduit/errors"
	"github.com/TheRebelOfBabylon/Conduit/intercept"
	e "github.com/pkg/errors"
	"github.com/rs/zerolog"
)

const (
	ErrLndNotFound = errors.Error("lnd command not found. Please install lnd to use conduit")
	ErrLndVersion  = errors.Error("lnd --version called. Gracefully exiting now...")
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
			// prevent panic conditions where we're looking at indices that don't exist
			if len(captures) == 0 {
				continue
			}
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
	// starting LND
	_, err := startLnd(cfg, wg, &log, shutdownInterceptor)
	if err != nil && err != ErrLndVersion {
		err = e.Wrap(err, "could not start lnd")
		log.Fatal().Msg(err.Error())
		return err
	} else if err == ErrLndVersion {
		return nil
	}
	<-shutdownInterceptor.ShutdownChannel()
	wg.Wait()
	return nil
}

// startLnd starts LND if it's been installed with a given config
func startLnd(cfg *Config, wg sync.WaitGroup, log *zerolog.Logger, shutdownInterceptor *intercept.Interceptor) (*bufio.Scanner, error) {
	// Let's check if LND is installed
	if _, err := exec.LookPath("lnd"); err != nil {
		log.Fatal().Msg(ErrLndNotFound.Error())
		return nil, ErrLndNotFound
	}
	// Check to see if we called -V, if so, we call lnd -V, display and exit
	if cfg.LndShowVersion {
		cmd := exec.Command("lnd", "--version")
		cmdReader, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal().Msg(fmt.Sprint(err))
			return nil, err
		}
		scanner := bufio.NewScanner(cmdReader)
		go func() {
			for scanner.Scan() {
				fmt.Println(scanner.Text())
				break
			}
		}()
		if err := cmd.Start(); err != nil {
			log.Fatal().Msg(fmt.Sprint(err))
			return scanner, err
		}
		if err := cmd.Wait(); err != nil {
			log.Fatal().Msg(fmt.Sprint(err))
			return scanner, err
		}
		return nil, ErrLndVersion
	}
	// We get all LND config from our config to pass
	// startup LND
	cmd := exec.Command("lnd", "--bitcoin.simnet", "--bitcoin.active", "--bitcoin.node=btcd")
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal().Msg(fmt.Sprint(err))
		return nil, err
	}
	scanner := bufio.NewScanner(cmdReader)
	re := regexp.MustCompile(lndLogRegex)
	wg.Add(1)
	go parseLndLog(scanner, log, re, shutdownInterceptor.ShutdownChannel(), wg)
	if err := cmd.Start(); err != nil {
		log.Fatal().Msg(fmt.Sprint(err))
		return scanner, err
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal().Msg(fmt.Sprint(err))
		return scanner, err
	}
	return scanner, nil
}
