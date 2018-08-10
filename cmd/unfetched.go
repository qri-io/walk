package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/datatogether/ffi"
	"github.com/qri-io/walk/lib"
	"github.com/spf13/cobra"
)

// UnfetchedURLsCmd is a command to check sitemaps for unfetched urls
var UnfetchedURLsCmd = &cobra.Command{
	Use:   "unfetched",
	Short: "list sitemap destination links that haven't been fetched",
	Long: `'sitemap crawl' currently misses a few links during it's process, this
command delivers a list of urls that *haven't* been fetched yet. Use unfetched
to ensure there's a record for every link (it should report 0 when the map is
as complete as can be), and add missing links to Seeds in configuration`,
	Args: cobra.MinimumNArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		format, err := cmd.Flags().GetString("format")
		if err != nil {
			panic(err.Error())
		}
		cfgPath, err := cmd.Flags().GetString("config")
		if err != nil {
			panic(err.Error())
		}
		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			panic(err.Error())
		}
		outputPath = fmt.Sprintf("%s.%s", outputPath, format)

		cfg := &lib.Config{}
		lib.JSONConfigFromFilepath(cfgPath)(cfg)

		// TODO - fix
		hosts := []*url.URL{}
		// hosts := make([]*url.URL, len(cfg.Domains))
		// for i, domain := range cfg.Domains {
		// 	u, err := url.Parse(domain)
		// 	if err != nil {
		// 		panic(err.Error())
		// 	}
		// 	hosts[i] = u
		// }

		data, err := ioutil.ReadFile(args[0])
		if err != nil {
			panic(err.Error())
		}

		urls := map[string]*lib.Resource{}
		if err := json.Unmarshal(data, &urls); err != nil {
			panic(err.Error())
		}

		f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}

		if format == "json" {
			f.Write([]byte{'['})
		}

		unfetched := 0
		checked := 0
		for _, u := range urls {
		LINKS:
			for _, l := range u.Links {
				checked++
				link, err := url.Parse(l)
				if err != nil {
					panic("error parsing url: " + err.Error())
				}

				for _, d := range hosts {
					if link.Host != d.Host {
						continue LINKS
					}
				}

				if err := isWebpageURL(link); err == nil {
					if _, ok := urls[l]; !ok {
						var data []byte
						switch format {
						case "txt":
							data = append([]byte(l), '\n')
						case "json":
							if unfetched == 0 {
								data = []byte(fmt.Sprintf("\n  \"%s\"", l))
							} else {
								data = []byte(fmt.Sprintf(",\n  \"%s\"", l))
							}
						}
						if _, err := f.Write(data); err != nil {
							panic(err)
						}
						unfetched++
					}
				}

			}
		}
		if format == "json" {
			f.Write([]byte{'\n', ']'})
		}
		f.Close()

		fmt.Printf("wrote %d/%d unfetched links to %s\n", unfetched, checked, outputPath)
	},
}

func init() {
	UnfetchedURLsCmd.Flags().StringP("config", "c", "sitemap.config.json", "path to configuration json file")
	UnfetchedURLsCmd.Flags().StringP("output", "o", "unfetched", "path to output without extension")
	UnfetchedURLsCmd.Flags().StringP("format", "f", "json", "output format. one of [txt,json]")
}

var htmlMimeTypes = map[string]bool{
	"text/html":                 true,
	"text/html; charset=utf-8":  true,
	"text/plain; charset=utf-8": true,
	"text/xml; charset=utf-8":   true,
}

// htmlExtensions is a dictionary of "file extensions" that normally contain
// html content
var htmlExtensions = map[string]bool{
	".asp":   true,
	".aspx":  true,
	".cfm":   true,
	".html":  true,
	".net":   true,
	".php":   true,
	".xhtml": true,
	".":      true,
	"":       true,
}

var invalidSchemes = map[string]bool{
	"data":   true,
	"mailto": true,
	"ftp":    true,
}

func isWebpageURL(u *url.URL) error {
	if invalidSchemes[u.Scheme] {
		return fmt.Errorf("invalid scheme: %s", u.Scheme)
	}

	filename, err := ffi.FilenameFromUrlString(u.String())
	if err != nil {
		return fmt.Errorf("ffi err: %s", err.Error())
	}

	ext := filepath.Ext(filename)
	if !htmlExtensions[ext] {
		return fmt.Errorf("non-webpage extension: %s", ext)
	}

	return nil
}
