package etcdlocker

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/coreos/etcd/client"
)

var (
	ErrMutexHasLocked = errors.New("muetx has locked")
)

type Mutex struct {
	client client.Client
	api    client.KeysAPI
	ttl    time.Duration
	key    string
	id     string
	logger Logger
	wg     sync.WaitGroup
	locker sync.RWMutex
	ttlCh  chan struct{}
}

func (m *Mutex) Lock() (err error) {
	m.locker.RLock()
	ch := m.ttlCh
	m.locker.RUnlock()
	if ch != nil {
		return ErrMutexHasLocked
	}

	opt := &client.SetOptions{
		PrevExist: client.PrevNoExist,
		TTL:       m.ttl + time.Second*5,
	}

	defer func() {
		if err == nil {
			m.locker.Lock()
			m.ttlCh = make(chan struct{})
			m.locker.Unlock()
			m.wg.Add(1)
			go m.keepAlive()
		}
	}()

	var resp *client.Response
	for {
		if _, err = m.api.Set(context.TODO(), m.key, m.id, opt); err == nil {
			return nil
		}

		etcdError, is := err.(client.Error)
		if !is {
			return err
		}

		if etcdError.Code != client.ErrorCodeNodeExist {
			return err
		}

		m.logger.Printf("wait the locker release %s", m.key)
		w := m.api.Watcher(m.key, nil)
		for {
			resp, err = w.Next(context.TODO())
			if err != nil {
				continue
			}

			if resp.Action == "delete" || resp.Action == "expire" {
				break
			}
		}
	}

	return nil
}

func (m *Mutex) Unlock() error {
	m.locker.Lock()
	close(m.ttlCh)
	m.wg.Wait()
	m.ttlCh = nil
	m.locker.Unlock()

	for {
		_, err := m.api.Delete(context.TODO(), m.key, nil)
		if err == nil {
			return nil
		}

		m.logger.Printf("delete locker %s failed %s", m.key, err)
		etcdError, is := err.(client.Error)
		if !is {
			continue
		}

		if etcdError.Code == client.ErrorCodeKeyNotFound {
			return nil
		}
	}
}

func (m *Mutex) keepAlive() {
	go func() {
		defer m.wg.Done()
		defer m.logger.Printf("stop %s ttl goroutine", m.key)
		m.logger.Printf("start %s ttl goroutine", m.key)

		for {
			select {
			case <-m.ttlCh:
				return
			case <-time.After(m.ttl):
				if _, err := m.api.Set(context.TODO(), m.key, m.id, &client.SetOptions{PrevValue: m.id, PrevExist: client.PrevExist, TTL: m.ttl + time.Second*5}); err == nil {
					m.logger.Printf("update %s id %s ttl ok", m.key, m.id)
				} else {
					m.logger.Printf("update %s id %s ttl failed %s", m.key, m.id, err.Error())
					return
				}
			}
		}
	}()
}

func New(key string, endpoints []string, opts ...Option) (*Mutex, error) {
	options := newOptions(opts...)

	conf := client.Config{
		Endpoints: endpoints,
		Transport: client.DefaultTransport,
	}

	c, err := client.New(conf)
	if err != nil {
		return nil, err
	}

	hn, _ := os.Hostname()

	m := &Mutex{
		client: c,
		api:    client.NewKeysAPI(c),
		ttl:    options.TTL,
		key:    key,
		id:     hn + "-" + time.Now().String(),
		logger: &nopLogger{},
	}

	return m, nil
}
