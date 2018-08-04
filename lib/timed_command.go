package lib

import (
	"net/url"
	"time"
)

// TimedCmd defines a Command implementation that sets an internal timestamp
// whenever it's URL method is called
type TimedCmd struct {
	U       *url.URL
	M       string
	Started time.Time
}

// NewTimedGet creates a new GET command with an internal Timer
func NewTimedGet(rawurl string) (*TimedCmd, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	return &TimedCmd{
		U: u,
		M: "GET",
	}, nil
}

// URL returns the resource targeted by this command.
func (c *TimedCmd) URL() *url.URL {
	if c.Started.IsZero() {
		c.Started = time.Now()
	}
	return c.U
}

// Method returns the HTTP verb to use to process this command (i.e. "GET", "HEAD", etc.).
func (c *TimedCmd) Method() string {
	return c.M
}
