package core

import (
	"github.com/TheRebelOfBabylon/Conduit/intercept"
	"github.com/rs/zerolog"
)

// Main is the true entry point for LND
func Main(shutdownInterceptor *intercept.Interceptor, cfg *Config, log zerolog.Logger) error {
	<-shutdownInterceptor.ShutdownChannel()
	return nil
}
