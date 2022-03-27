package main

import (
	"fmt"
	"os"

	"github.com/TheRebelOfBabylon/Conduit/core"
	"github.com/TheRebelOfBabylon/Conduit/intercept"
)

// main is the entry point for Conduit
func main() {
	// Hook interceptor for os signals.
	shutdownInterceptor, err := intercept.InitInterceptor()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	config, err := core.InitConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log, err := core.InitLogger(config)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	shutdownInterceptor.Logger = &log
	if err = core.Main(shutdownInterceptor, config, log); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
