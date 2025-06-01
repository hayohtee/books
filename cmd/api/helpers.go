package main

import "fmt"

// background is a helper method for running background
// tasks.
// It accepts arbitrary function as parameter and run it
// in a goroutine, it also recovers any panic when calling
// the function.
func (app *application) background(fn func()) {
	// Increment the WaitGroup counter.
	app.wg.Add(1)

	// Launch background goroutine.
	go func() {
		// Use defer to decrement the WaitGroup counter before goroutine returns.
		defer app.wg.Done()

		// Recover any panic.
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Sprintf("%v", err))
			}
		}()

		// Execute the function.
		fn()
	}()
}
