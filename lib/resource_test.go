package lib

import (
	"time"
)

func exampleResourceA() *Resource {
	return &Resource{
		URL:       "https://www.a.com",
		Timestamp: time.Date(2000, 0, 0, 0, 0, 0, 0, time.UTC),
		Status:    200,
		Links: []string{
			"https://www.a.com/a",
			"https://www.a.com/b",
		},
	}
}

func exampleResourceAa() *Resource {
	return &Resource{
		URL:       "https://www.a.com/a",
		Timestamp: time.Date(2000, 0, 0, 0, 0, 0, 0, time.UTC),
		Status:    200,
		Links: []string{
			"https://www.a.com",
		},
	}
}
