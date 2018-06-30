package etcdlock

import "time"

type Options struct {
	TTL time.Duration
}

type Option func(*Options)

func TTL(d time.Duration) Option {
	return func(options *Options) {
		options.TTL = d
	}
}

func newOptions(opts ...Option) (sopt Options) {
	for _, f := range opts {
		f(&sopt)
	}

	if sopt.TTL == 0 {
		sopt.TTL = time.Second * 5
	}

	return
}
