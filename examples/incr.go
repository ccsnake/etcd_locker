package main

import (
	"etcd_locker"
	"fmt"

	"github.com/juju/errors"
)

func main() {
	etcd := []string{"http://192.168.11.204:2379"}

	var count int
	c := 1

	ch := make(chan error, c)

	for i := 0; i < c; i++ {
		go func() { ch <- Add(&count, 1, 100, etcd) }()
	}

	for idx := 0; idx < c; idx++ {
		if err := <-ch; err != nil {
			fmt.Printf("add fault for %s\n", err.Error())
		}
	}
	fmt.Printf("inc test done,result %d\n", count)
}

func Add(val *int, inc, times int, etcdAdders []string) error {
	m, err := etcdlock.New("TEST:LOCK:1", etcdAdders)
	if err != nil {
		return err
	}

	for idx := 0; idx < times; idx++ {
		if err := m.Lock(); err != nil {
			return errors.Annotatef(err, "lock %d", idx)
		}

		*val += inc
		if err := m.Unlock(); err != nil {
			return errors.Annotatef(err, "unlock %d", idx)

		}
	}
	return nil
}
