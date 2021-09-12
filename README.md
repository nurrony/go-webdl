# Simple File Downloader over HTTP/HTTPS (webdl)

[![Go Report Card](https://goreportcard.com/badge/github.com/nmrony/go-webdl)](https://goreportcard.com/report/github.com/nmrony/go-webdl)

Resumable, Concurrent and Simple file downloader using Golang (fun side project)

## Features

- Resumable
- Parallel download
- Visual Progressbar

## Installation

1. Go to release page and download exectutable for your OS.
2. Rename it to `webdl`
3. Set it to your path

## Usage

```sh
webdl -url <file-url> [-o={output-filename}] [-n=1] [-resume=false] [-buffer=1KiB]

# Example
webdl -url https://www.hq.nasa.gov/alsj/a17/A17_FlightPlan.pdf
```

## Preview

![preview](assets/demo.gif)
