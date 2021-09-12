package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/nmrony/go-webdl/pkg/config"
	"github.com/nmrony/go-webdl/pkg/downloader"
)

func usage() {
	fmt.Print("\nUsage of webdl:\n\n")
	fmt.Print("webdl -url <file-url> [-o={output-filename}] [-n=1] [-resume=false] [-buffer=1024]\n\n")
	flag.PrintDefaults()
}

func main() {
	url := flag.String("url", "", "Download url (required)")
	concurrency := flag.Int("n", 1, "Concurrency level")
	filename := flag.String("o", "", "Output file name")
	bufferSize := flag.Int("buffer", 1*1024, "The buffer size in KiB to copy from response body")
	resume := flag.Bool("resume", false, "Resume the download (default false)")

	flag.Usage = usage

	flag.Parse()

	if *url == "" {
		log.Fatal("Please specify the url using -url parameter")
	}

	config := &config.Config{
		Url:            *url,
		Concurrency:    *concurrency,
		OutFilename:    *filename,
		CopyBufferSize: *bufferSize,
		Resume:         *resume,
	}

	d, err := downloader.NewFromConfig(config)

	if err != nil {
		log.Fatal(err.Error())
	}

	termCh := make(chan os.Signal)
	signal.Notify(termCh, os.Interrupt)
	go func() {
		<-termCh
		println("\nExiting ...")
		d.Pause()
	}()

	d.Download()
	if d.Paused {
		println("\nDownload has paused. Resume it again with -resume=true parameter.")
	} else {
		println("Download completed.")
	}
}
