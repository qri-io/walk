package lib

import (
	"fmt"
)

var (
	// ErrNotFound is a standard error for a missing thing
	ErrNotFound = fmt.Errorf("not found")
)

// Queue is an interface for queing up Requests. it's expected that the queue
// operates in FIFO order, pushing requests that need processing onto one end
// popping requests off the other for processing
type Queue interface {
	Push(*Request)
	Pop() *Request
	Len() (int, error)
	Chan() (chan *Request, error)
}

// MemQueue is an in-memory implementation of the Queue interface,
// it's just a fancy channel
type MemQueue chan *Request

// Push adds a fetch request to the end of the queue
func (q MemQueue) Push(t *Request) {
	q <- t
}

// Pop removes a request from the queue
// TODO - consider implementing acknowledgement/confirmation for guaranteed delivery:
// when popping, move the item to a secondary queue, then delete it from that queue when
// acknowledgement happens or move it back to the main queue if you don’t get
// acknowledgement within a given timeframe because the worker died.
func (q MemQueue) Pop() *Request {
	return <-q
}

// Len returns the number of Requests in the queue
func (q MemQueue) Len() (int, error) {
	return len(q), nil
}

// Chan returns the queue structured as a go channel
func (q MemQueue) Chan() (chan *Request, error) {
	return (chan *Request)(q), nil
}
