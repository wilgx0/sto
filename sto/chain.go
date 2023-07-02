package sto

import (
	"fmt"
	"runtime"
)

type JobWrapper func(Job, interface{}) Job

type Chain struct {
	wrappers []JobWrapper
}

func NewChain(c ...JobWrapper) Chain {
	return Chain{c}
}

func (c Chain) Then(j Job, payload interface{}) Job {
	for i := range c.wrappers {
		j = c.wrappers[len(c.wrappers)-i-1](j, payload)
	}
	return j
}

func Recover(logger Logger) JobWrapper {
	return func(j Job, payload interface{}) Job {
		return FuncJob(func(payload interface{}) {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					logger.Error(err, "panic", "stack", "...\n"+string(buf))
				}
			}()
			j.Run(payload)
		})

	}
}
