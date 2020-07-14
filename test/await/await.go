package await

import (
	"fmt"
	"time"
)

type AwaitData struct {
	initialDelay, interval, timeout time.Duration
}

func newDefault() AwaitData {
	return AwaitData{
		initialDelay: 0 * time.Millisecond,
		interval:     100 * time.Millisecond,
		timeout:      5 * time.Second,
	}
}

func After(initialDelay time.Duration) AwaitData {
	return newDefault().After(initialDelay)
}

func (a AwaitData) After(initialDelay time.Duration) AwaitData {
	a.initialDelay = initialDelay
	return a
}

func AtMost(timeout time.Duration) AwaitData {
	return newDefault().AtMost(timeout)
}

func (a AwaitData) AtMost(timeout time.Duration) AwaitData {
	a.timeout = timeout
	return a
}

func Every(interval time.Duration) AwaitData {
	return newDefault().Every(interval)
}

func (a AwaitData) Every(interval time.Duration) AwaitData {
	a.interval = interval
	return a
}

func That(runner func() error) error {
	return newDefault().That(runner)
}

func (a AwaitData) That(runner func() error) error {
	timeout := make(chan struct{})
	go func() {
		<-time.After(a.timeout)
		close(timeout)
	}()
	errors := make(chan error)
	<-time.After(a.initialDelay)
	go func() {
		defer close(errors)
		var lastErr error
		for {
			select {
			case <-timeout:
				errors <- fmt.Errorf("timed out waiting for condition: %w", lastErr)
				return
			case <-time.After(a.interval):
				err := runner()
				if err == nil {
					return
				}
				lastErr = err
			}
		}
	}()
	return <-errors
}
