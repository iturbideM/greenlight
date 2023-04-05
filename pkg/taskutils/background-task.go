package taskutils

import (
	"fmt"
	"sync"
)

var wg = sync.WaitGroup{}

type Logger interface {
	PrintError(err error, properties map[string]string)
}

// Setup Background task with with recover, and assures graceful exit on program exit
// when calling WaitAll()
//
// WARNING: This function by itself does not use a separete goroutine. Use `go` to run it concurrently
//
// Technically a little bit slower than using a goroutine, if you know it will not panic and don't care about
// a graceful exit, you should use a goroutine.
func BackgroundTask(logger Logger, task func()) {
	wg.Add(1)
	defer func() {
		defer wg.Done()
		if err := recover(); err != nil {
			logger.PrintError(fmt.Errorf("%s", err), nil)
		}
	}()

	task()
}

// Block until all background tasks are finished
func WaitAll() {
	wg.Wait()
}
