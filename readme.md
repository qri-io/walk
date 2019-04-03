# Walk

[![GoDoc](https://godoc.org/github.com/qri-io/walk?status.svg)](http://godoc.org/github.com/qri-io/walk) [![License](https://img.shields.io/github/license/qri-io/walk.svg?style=flat-square)](./LICENSE) [![Codecov](https://img.shields.io/codecov/c/github/qri-io/walk.svg?style=flat-square)](https://codecov.io/gh/qri-io/walk) [![CI](https://img.shields.io/circleci/project/github/qri-io/walk.svg?style=flat-square)](https://circleci.com/gh/qri-io/walk)

## What is this project?
Walk is a sitemapping tool. It's designed to crawl and snapshot sitemaps from URLs in order to understand how sites change over time.

**Current status:** Design & prototyping phase

**Key user:** EDGI web monitoring analysts, who want to understand changes in the structure of sites, and particularly look out for "islanding" (when a web resource still exists at a URL but is no longer linked to from anywhere)

**How to get involved:**
* Reach out over [Slack](https://archivers-slack.herokuapp.com/) or
* Make an issue on this repo to explain your interest or use case

---
## Building from source
To build Walk you'll need the [go programming language](https://golang.org) on your machine.

```shell
$ go get github.com/qri-io/walk
$ cd $GOPATH/src/github.com/qri-io/walk
$ make install
```
---

## Architecture

### Minimum viable version
Walk is written as a library accessed through a command line interface (CLI). It takes a configuration file and returns a sitemap in NDJSON (newline-delimited JSON).

#### Basic architecture
There are two major components of Walk:
1. **Crawler**: takes a URL and spits out linked URLs
1. **Coordinator/result handler**: orchestrates and takes the output of crawls, maintains global state

The crawler crawls just one level (links from one particular page) at a time. The results are then sent to the coordinator/result handler, which performs some minimal processing (e.g. de-duplication of lists of returned URLs) and queues the next iterations of crawling.

![Very small/blurry diagram of the jobs, workers, and results interacting with the coordinator](https://lh3.googleusercontent.com/W2HUfQXgmz-MQQfc7Z1QpeAo1dS7WqAZ8AziG-uYYVZu_GRCjXDIhgIuqcV9WVp8cFMtV-IAjWsRMqohxSWPPHowLXZritm8QHqwLiAltDBxqodZX54Ogo3mkOmYCUKgGADN5hjx8RvCbh8-adJfqjRZkmCFjNgVRFha_5V-pa9PiSTAGkB3I2GphzQT9qJ4GDgplogezKWYX43kWzszcsofEHhkow4LPvV7xVYHh3HSo-AbKtKdD8q_ZAqO2gmn2h_2HXv8SPqqjH2Rbft3GwHcIPrSaFqInwhryzSakkHLrDA1R6AcC_9CxGh3t_fCvJSPmuxMK15dzuWnauoD2MlAcS_AlhOmtQFP7p-zW-OWpzqgMlYtnMYYrcXWefycnaf-Bv46sU_xGae_-ex3yk2TgPCmSwVVx8l3EK7aiSDYM0rqGCZS6w5SXYBlTVXnn4XYv2qToabUX0IDFlopuvMmzCiAg_gzNeou0_Q_8Zx56cK2yMnJkJp9G_oHu-ndQ2hJRTD4dcV0LcjQoFwZ86jurcTcjc0mpv-4caHgxDEzXQVgzscCwhSvRLHGcjxZNghOSFknXSOIqCmVVB_c9EQv3K4scHXliJkYfyr00CRAYG-UeoDWhylWTETMpktWnlt0VaOTTrn6sVfuueMFIBGGxE6jY_wq=w317-h99-no)

#### Interaction
Walk's coordinator runs on a cloud server in order to minimize potential disruption of a job.

Jobs can be queued to the server through a REST API (http) or through the `server` flag in the CLI.

### Set in a broader vision

Walk is a modular system of components. It begins as a crawler/sitemapper, building snapshots of URL links.

From that foundation, it should be able to grow into a web scraper (takes URL and gives list of resources mapped to content).

The grandest vision of Walk is a tool that can take in a BIG list of URLs (30k+), reliably snapshot both sitemaps and webpage resources, and export the results into a number of formats. It is intended as a shared resource for many projects.

System diagram:

![System Diagram](docs/system_diagram.jpg)

(Diagram notes: System components are in square-edged boxes, while the format of data being sent between them is in round-edged boxes. Open questions highlighted in green.)




