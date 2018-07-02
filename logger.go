package etcdlocker

import "log"

type Logger interface {
	Printf(fmt string, arg ...interface{})
}

type nopLogger struct {
}

func (*nopLogger) Printf(fmt string, arg ...interface{}) {
}

type StdLogger struct {
}

func (*StdLogger) Printf(fmt string, arg ...interface{}) {
	log.Printf(fmt, arg...)
}
