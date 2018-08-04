package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	ignore  = "http://epa.gov/newsreleases/search/"
	logFile = "sitemap.cleaned.log"
)

// CleanCmd cleans sitemap files
var CleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "clean a sitemap file",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := ioutil.ReadFile("sitemap.json")
		if err != nil {
			panic(err.Error())
		}

		urls := map[string]interface{}{}
		if err := json.Unmarshal(data, &urls); err != nil {
			panic(err.Error())
		}

		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}

		total := 0
		logged := 0
		for _, v := range urls {
			if url, ok := v.(map[string]interface{}); ok {
				total++
				if strings.HasPrefix(url["url"].(string), ignore) {
					continue
				}

				data, err := json.Marshal(v)
				if err != nil {
					panic(err.Error())
				}

				if n, err := f.Write(append(data, '\n')); err != nil {
					panic(err.Error())
				} else if n < len(data) {
					panic(fmt.Sprintf("%d < %d", n, len(data)))
				}
				logged++
			} else {
				panic(v)
			}
		}
		f.Close()

		cleaned, err := os.Open(logFile)
		if err != nil {
			panic(err.Error())
		}

		s := bufio.NewScanner(cleaned)
		s.Buffer(make([]byte, 0, 500*1024), 500*1024)
		s.Split(bufio.ScanLines)

		urls = map[string]interface{}{}
		scanned := 0
		for s.Scan() {
			data := s.Bytes()
			url := map[string]interface{}{}
			if err := json.Unmarshal(data, &url); err != nil {
				panic(err.Error())
			}
			urlstr := url["url"].(string)
			delete(url, "url")
			urls[urlstr] = url
			scanned++
		}
		if err := s.Err(); err != nil {
			panic(err)
		}

		fmt.Printf(`total:   %d
logged:   %d
removed:  %d
scanned:  %d
`, total, logged, total-logged, scanned)

		data, err = json.MarshalIndent(urls, "", "  ")
		if err != nil {
			panic(err.Error())
		}

		if err := ioutil.WriteFile("sitemap.cleaned.json", data, 0622); err != nil {
			panic(err.Error())
		}

	},
}
