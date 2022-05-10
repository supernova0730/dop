package zap

import (
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const callerSkip = 1

type St struct {
	l  *zap.Logger
	sl *zap.SugaredLogger
}

func New(level string, dev bool) *St {
	var cfg zap.Config

	if dev {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()

		switch level {
		case "error":
			cfg.Level.SetLevel(zap.ErrorLevel)
		case "warn": // default
			cfg.Level.SetLevel(zap.WarnLevel)
		case "info":
			cfg.Level.SetLevel(zap.InfoLevel)
		case "debug":
			cfg.Level.SetLevel(zap.DebugLevel)
		default:
			cfg.Level.SetLevel(zap.WarnLevel)
		}
	}

	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	l, err := cfg.Build(zap.AddCallerSkip(callerSkip))
	if err != nil {
		log.Fatal(err)
	}

	return &St{
		l:  l,
		sl: l.Sugar(),
	}
}

func (o *St) Fatal(args ...interface{}) {
	o.sl.Fatal(args...)
}

func (o *St) Fatalf(tmpl string, args ...interface{}) {
	o.sl.Fatalf(tmpl, args...)
}

func (o *St) Fatalw(msg string, err interface{}, args ...interface{}) {
	args = append(args, "error", err)
	o.sl.Fatalw(msg, args...)
}

func (o *St) Error(args ...interface{}) {
	o.sl.Error(args...)
}

func (o *St) Errorf(tmpl string, args ...interface{}) {
	o.sl.Errorf(tmpl, args...)
}

func (o *St) Errorw(msg string, err interface{}, args ...interface{}) {
	args = append(args, "error", err)
	o.sl.Errorw(msg, args...)
}

func (o *St) Warn(args ...interface{}) {
	o.sl.Warn(args...)
}

func (o *St) Warnf(tmpl string, args ...interface{}) {
	o.sl.Warnf(tmpl, args...)
}

func (o *St) Warnw(msg string, args ...interface{}) {
	o.sl.Warnw(msg, args...)
}

func (o *St) Info(args ...interface{}) {
	o.sl.Info(args...)
}

func (o *St) Infof(tmpl string, args ...interface{}) {
	o.sl.Infof(tmpl, args...)
}

func (o *St) Infow(msg string, args ...interface{}) {
	o.sl.Infow(msg, args...)
}

func (o *St) Debug(args ...interface{}) {
	o.sl.Debug(args...)
}

func (o *St) Debugf(tmpl string, args ...interface{}) {
	o.sl.Debugf(tmpl, args...)
}

func (o *St) Debugw(msg string, args ...interface{}) {
	o.sl.Debugw(msg, args...)
}

func (o *St) Sync() {
	if err := o.sl.Sync(); err != nil {
		log.Println("Fail to sync zap-logger", err)
	}
}
