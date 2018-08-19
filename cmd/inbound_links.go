package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/qri-io/walk/lib"
	"github.com/spf13/cobra"
)

// InboundLinksCmd is the command for listing links to a given url
var InboundLinksCmd = &cobra.Command{
	Use:   "inbound-links",
	Short: "output links to a given url to a file",
	Example: `  write all urls in "sitemap.json" that link to http://example.com:
  $ walk inbound-links sitemap.json http://example.com`,

	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var (
			u   *url.URL
			err error
		)

		path := args[0]
		rawurl := args[1]
		writepath, err := cmd.Flags().GetString("output")
		if err != nil {
			fmt.Printf("error getting flag: %s\n", err.Error())
			return
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("error reading sitemap file %s: %s", path, err.Error())
			return
		}

		u, err = url.Parse(rawurl)
		if err != nil {
			fmt.Printf("error parsing url:\n\t%s\n\t%s", rawurl, err.Error())
			return
		}
		urlstr := lib.NormalizeURLString(u)

		urls := map[string]*lib.Resource{}
		if err := json.Unmarshal(data, &urls); err != nil {
			fmt.Printf("error decoding JSON sitemap: %s", err.Error())
			return
		}

		inbound := []string{}

		checked := 0
		found := 0
		for ustr, urlinfo := range urls {
			checked++
			for _, l := range urlinfo.Links {
				if urlstr == l {
					found++
					inbound = append(inbound, ustr)
					break
				}
			}
		}

		linkdata, err := json.MarshalIndent(inbound, "", "  ")
		if err != nil {
			fmt.Printf("error encoding links list to json: %s\n", err.Error())
			return
		}

		if err := ioutil.WriteFile("inbond_links.json", linkdata, 0667); err != nil {
			fmt.Printf("error writing file to json: %s\n", err.Error())
			return
		}

		fmt.Printf("found %d/%d inbound links for %s\n", found, checked, rawurl)
		fmt.Printf("links written to %s\n", writepath)
	},
}

func init() {
	InboundLinksCmd.Flags().StringP("output", "o", "inbound_links.json", "path to write file to")
}
