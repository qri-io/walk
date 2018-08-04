package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/qri-io/walk/lib"
	"github.com/spf13/cobra"
)

// Stats is for calculating sitemap statistics
type Stats struct {
	URLCount    int
	AvgNumLinks float32
	AvgPageSize float32

	FirstFetch time.Time
	LastFetch  time.Time

	HostURLCount map[string]int
	StatusCount  map[int]int

	size  int64
	links int
}

// StatsCmd is the comand line command for calculating stats
var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "calculate stats on a sitemap",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// path := "sitemap.json"
		path := args[0]
		data, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("error reading sitemap file %s: %s", path, err.Error())
			return
		}

		urls := map[string]*lib.URL{}
		if err := json.Unmarshal(data, &urls); err != nil {
			fmt.Printf("error decoding JSON sitemap: %s", err.Error())
			return
		}

		stats := &Stats{
			URLCount: len(urls),
		}

		defer func() {
			data, err := json.MarshalIndent(stats, "", "  ")
			if err != nil {
				fmt.Printf("error encoding stats to JSON: %s", err.Error())
				return
			}

			fmt.Println(string(data))
		}()

		if len(urls) == 0 {
			return
		}

		stats.HostURLCount = map[string]int{}
		stats.StatusCount = map[int]int{}
		stats.FirstFetch = time.Date(3000, 0, 0, 0, 0, 0, 0, time.UTC)

		for rawurl, u := range urls {
			if u.Timestamp.Before(stats.FirstFetch) {
				stats.FirstFetch = u.Timestamp
			} else if u.Timestamp.After(stats.LastFetch) {
				stats.LastFetch = u.Timestamp
			}

			stats.links += len(u.Links)
			stats.size += u.ContentLength

			stats.StatusCount[u.Status]++
			if uu, err := url.Parse(rawurl); err == nil {
				stats.HostURLCount[uu.Host]++
			}
		}

		stats.AvgNumLinks = float32(stats.links) / float32(len(urls))
		stats.AvgPageSize = float32(stats.size) / float32(len(urls))

	},
}
