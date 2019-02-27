package lib

import (
	"time"
)

// Request is a URL that needs to be turned into a resource through
// Fetching (request the URL & recording the response) Requests are
// held in stores, placed in queues, and consumed by workers
type Request struct {
	JobID  string
	URL    string
	Status RequestStatus
	// TODO - currently not in use
	FetchAfter    time.Time
	AttemptsMade  int
	PrevResStatus int
}

// RequestStatus enumerates all possible states a request can be in
type RequestStatus int

const (
	// RequestStatusUnknown is the default state
	RequestStatusUnknown RequestStatus = iota
	// RequestStatusFetch indicates this Request still needs fetching
	RequestStatusFetch
	// RequestStatusQueued indicates this Request is queued for fetching
	RequestStatusQueued
	// RequestStatusRequesting indicates this Request is currently being fetched
	RequestStatusRequesting
	// RequestStatusDone indicates this Request has successfully completed
	RequestStatusDone
	// RequestStatusFailed indicates this request cannot be completed
	RequestStatusFailed
)
