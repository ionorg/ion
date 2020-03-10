package async

// Error .
type Error struct {
	Code   int
	Reason string
}

// Future .
type Future struct {
	c      chan struct{}
	result map[string]interface{}
	err    *Error
}

// NewFuture .
func NewFuture() *Future {
	future := Future{
		c:      make(chan struct{}, 1),
		result: make(map[string]interface{}),
		err:    nil,
	}
	return &future
}

// Await .
func (future *Future) Await() (map[string]interface{}, *Error) {
	<-future.c
	return future.result, future.err
}

// Then .
func (future *Future) Then(resolve func(result map[string]interface{}), reject func(err *Error)) {
	go func() {
		<-future.c
		if future.err != nil {
			reject(future.err)
		} else {
			resolve(future.result)
		}
	}()
}

// Resolve .
func (future *Future) Resolve(result map[string]interface{}) {
	future.result = result
	close(future.c)
}

// Reject .
func (future *Future) Reject(err *Error) {
	future.err = err
	close(future.c)
}
