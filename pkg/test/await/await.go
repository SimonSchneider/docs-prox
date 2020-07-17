package await

import (
	"fmt"
	"time"
)

//Config contains data for the await calls
type Config struct {
	initialDelay, interval, timeout time.Duration
}

func newDefault() Config {
	return Config{
		initialDelay: 0 * time.Millisecond,
		interval:     100 * time.Millisecond,
		timeout:      5 * time.Second,
	}
}

func (c Config) clone() Config {
	return Config{
		initialDelay: c.initialDelay,
		interval:     c.interval,
		timeout:      c.timeout,
	}
}

// After sets the initial delay of when to evaluate the function
func After(initialDelay time.Duration) Config {
	return newDefault().After(initialDelay)
}

// After sets the initial delay of when to evaluate the function
func (c Config) After(initialDelay time.Duration) Config {
	n := c.clone()
	n.initialDelay = initialDelay
	return n
}

// AtMost sets the timeout after which the function is deemed failed
func AtMost(timeout time.Duration) Config {
	return newDefault().AtMost(timeout)
}

// AtMost sets the timeout after which the function is deemed failed
func (c Config) AtMost(timeout time.Duration) Config {
	n := c.clone()
	n.timeout = timeout
	return n
}

// Every sets the interval at which to test the function
func Every(interval time.Duration) Config {
	return newDefault().Every(interval)
}

// Every sets the interval at which to test the function
func (c Config) Every(interval time.Duration) Config {
	n := c.clone()
	n.interval = interval
	return n
}

// That runs the passed function and returns error if it fails after the given timeout
func That(runner func() error) error {
	return newDefault().That(runner)
}

// That runs the passed function and returns error if it fails after the given timeout
func (c Config) That(runner func() error) error {
	timeout := make(chan struct{})
	go func() {
		<-time.After(c.timeout)
		close(timeout)
	}()
	errors := make(chan error)
	<-time.After(c.initialDelay)
	go func() {
		defer close(errors)
		var lastErr error
		for {
			select {
			case <-timeout:
				errors <- fmt.Errorf("timed out waiting for condition: %w", lastErr)
				return
			case <-time.After(c.interval):
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
