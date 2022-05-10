package logger

type Full interface {
	Debug(args ...interface{})
	Debugf(tmpl string, args ...interface{})
	Debugw(msg string, args ...interface{})

	Info(args ...interface{})
	Infof(tmpl string, args ...interface{})
	Infow(msg string, args ...interface{})

	Warn(args ...interface{})
	Warnf(tmpl string, args ...interface{})
	Warnw(msg string, args ...interface{})

	Error(args ...interface{})
	Errorf(tmpl string, args ...interface{})
	Errorw(msg string, err interface{}, args ...interface{})

	Fatal(args ...interface{})
	Fatalf(tmpl string, args ...interface{})
	Fatalw(msg string, err interface{}, args ...interface{})
}

type Lite interface {
	Infow(msg string, args ...interface{})
	Warnw(msg string, args ...interface{})
	Errorw(msg string, err interface{}, args ...interface{})
}

type WarnAndError interface {
	Warnw(msg string, args ...interface{})
	Errorw(msg string, err interface{}, args ...interface{})
}
