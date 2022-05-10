package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/rendau/dop/adapters/logger"
)

const (
	ReadHeaderTimeout = 10 * time.Second
	ReadTimeout       = 2 * time.Minute
	MaxHeaderBytes    = 300 * 1024
)

type St struct {
	lg logger.Lite

	addr   string
	server *http.Server
	eChan  chan error
}

func New(
	lg logger.Lite,
) *St {
	return &St{lg: lg}
}

func (a *St) Start(addr string, handler http.Handler) <-chan error {
	a.addr = addr

	a.server = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: ReadHeaderTimeout,
		ReadTimeout:       ReadTimeout,
		MaxHeaderBytes:    MaxHeaderBytes,
	}

	a.eChan = make(chan error, 1)

	a.lg.Infow("Start rest-api", "addr", a.server.Addr)

	go func() {
		err := a.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			a.lg.Errorw("Http server closed", err)
			a.eChan <- err
		}
	}()

	return a.eChan
}

func (a *St) Shutdown(timeout time.Duration) bool {
	defer close(a.eChan)

	ctx, ctxCancel := context.WithTimeout(context.Background(), timeout)
	defer ctxCancel()

	err := a.server.Shutdown(ctx)
	if err != nil {
		a.lg.Errorw("Fail to shutdown http-api", err, "addr", a.addr)
		return false
	}

	return true
}
