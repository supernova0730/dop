package https

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	cors "github.com/rs/cors/wrapper/gin"
	"github.com/supernova0730/dop/adapters/logger"
	"github.com/supernova0730/dop/dopErrs"
	"github.com/supernova0730/dop/dopTypes"
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

func Start(addr string, handler http.Handler, lg logger.Lite) *St {
	s := &St{
		lg:   lg,
		addr: addr,
		server: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: ReadHeaderTimeout,
			ReadTimeout:       ReadTimeout,
			MaxHeaderBytes:    MaxHeaderBytes,
		},
		eChan: make(chan error, 1),
	}

	s.lg.Infow("Start rest-api", "addr", s.server.Addr)

	go func() {
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			s.lg.Errorw("Http server closed", err)
			s.eChan <- err
		}
	}()

	return s
}

func (s *St) Wait() <-chan error {
	return s.eChan
}

func (s *St) Shutdown(timeout time.Duration) bool {
	defer close(s.eChan)

	ctx, ctxCancel := context.WithTimeout(context.Background(), timeout)
	defer ctxCancel()

	err := s.server.Shutdown(ctx)
	if err != nil {
		s.lg.Errorw("Fail to shutdown http-api", err, "addr", s.addr)
		return false
	}

	return true
}

func Error(c *gin.Context, err error) bool {
	if err != nil {
		_ = c.Error(err)
		return true
	}
	return false
}

func GetAuthToken(c *gin.Context) string {
	token := c.GetHeader("Authorization")

	if token == "" { // try from query parameter
		token = c.Query("auth_token")
	} else {
		if strings.HasPrefix(token, "Bearer ") {
			token = token[7:]
		}
	}

	return token
}

func BindJSON(c *gin.Context, obj any) bool {
	err := c.ShouldBindJSON(obj)
	if err != nil {
		Error(c, dopErrs.ErrWithDesc{
			Err:  dopErrs.BadJson,
			Desc: err.Error(),
		})

		return false
	}

	return true
}

func BindQuery(c *gin.Context, obj any) bool {
	err := c.ShouldBindQuery(obj)
	if err != nil {
		Error(c, dopErrs.ErrWithDesc{
			Err:  dopErrs.BadQueryParams,
			Desc: err.Error(),
		})

		return false
	}

	return true
}

func MwRecovery(lg logger.WarnAndError, handler func(*gin.Context, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			var err error

			if gErr := c.Errors.Last(); gErr != nil { // gin error
				if gErr.IsType(gin.ErrorTypeBind) {
					err = dopErrs.ErrWithDesc{
						Err:  dopErrs.BadJson,
						Desc: err.Error(),
					}
				} else {
					err = gErr.Err
				}
			} else if recoverRep := recover(); recoverRep != nil { // recovery error
				var ok bool
				if err, ok = recoverRep.(error); !ok {
					err = errors.New(fmt.Sprint(recoverRep))
				}
			}

			if err == nil {
				return
			}

			if handler != nil {
				handler(c, err)
				return
			}

			switch cErr := err.(type) {
			case dopErrs.Err:
				c.AbortWithStatusJSON(http.StatusBadRequest, dopTypes.ErrRep{
					ErrorCode: cErr.Error(),
				})
			case dopErrs.ErrWithDesc:
				c.AbortWithStatusJSON(http.StatusBadRequest, dopTypes.ErrRep{
					ErrorCode: cErr.Err.Error(),
					Desc:      cErr.Desc,
				})
			case dopErrs.FormErr:
				fields := map[string]string{}

				for k, v := range cErr.Fields {
					fields[k] = v.Error()
				}

				c.AbortWithStatusJSON(http.StatusBadRequest, dopTypes.ErrRep{
					ErrorCode: dopErrs.FormValidate.Error(),
					Fields:    fields,
				})
			default:
				lg.Errorw(
					"Error in httpc handler",
					err,
					"method", c.Request.Method,
					"path", c.Request.URL.String(),
				)

				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		c.Next()
	}
}

func MwCors() gin.HandlerFunc {
	return cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool { return true },
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodConnect,
			http.MethodOptions,
			http.MethodTrace,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           604800,
	})
}
