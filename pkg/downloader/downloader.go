package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/nmrony/go-webdl/internal/utils"
	"github.com/nmrony/go-webdl/pkg/config"
	"github.com/schollz/progressbar/v3"
)

type downloader struct {
	Paused      bool
	config      *config.Config
	context     context.Context
	cancel      context.CancelFunc
	progressBar *progressbar.ProgressBar
}

// Returns downloader for a given config
func NewFromConfig(config *config.Config) (*downloader, error) {
	if config.Url == "" {
		return nil, errors.New("url is empty")
	}
	if config.Concurrency < 1 {
		config.Concurrency = 1
		fmt.Print("Concurrency level: 1")
	}
	if config.OutFilename == "" {
		config.OutFilename = path.Base(config.Url)
	}
	if config.CopyBufferSize == 0 {
		config.CopyBufferSize = 1024
	}

	dl := &downloader{config: config}

	// rename file if such file already exist
	dl.renameFilenameIfNecessary()
	fmt.Printf("Output file: %s\n", filepath.Base(config.OutFilename))
	return dl, nil
}

// Returns downloader for a default config
func New(url string) (*downloader, error) {
	if url == "" {
		return nil, errors.New("url is empty")
	}

	config := &config.Config{
		Url:         url,
		Concurrency: 1,
	}

	return NewFromConfig(config)
}

// Pause a download
func (dl *downloader) Pause() {
	dl.Paused = true
	dl.cancel()
}

// Resume a download
func (dl *downloader) Resume() {
	dl.config.Resume = true
	dl.Paused = false
	dl.Download()
}

// Returns the progress bar's state
func (dl *downloader) ProgressState() progressbar.State {
	if dl.progressBar != nil {
		return dl.progressBar.State()
	}

	return progressbar.State{}
}

// Add a number to the filename if file already exist
// For instance, if filename `hello.pdf` already exist
// it returns hello(1).pdf
func (dl *downloader) renameFilenameIfNecessary() {
	if dl.config.Resume {
		return // in resume mode, no need to rename
	}

	if _, err := os.Stat(dl.config.OutFilename); err == nil {
		counter := 1
		filename, ext := utils.GetFilenameAndExt(dl.config.OutFilename)
		outDir := filepath.Dir(dl.config.OutFilename)

		for err == nil {
			fmt.Printf("File %s%s already exist\n", filename, ext)
			newFilename := fmt.Sprintf("%s(%d)%s", filename, counter, ext)
			dl.config.OutFilename = path.Join(outDir, newFilename)
			_, err = os.Stat(dl.config.OutFilename)
			counter += 1
		}
	}
}

func (dl *downloader) getPartFilename(partNum int) string {
	return dl.config.OutFilename + ".part" + strconv.Itoa(partNum)
}

// Start download of a file
func (dl *downloader) Download() {
	ctx, cancel := context.WithCancel(context.Background())
	dl.context = ctx
	dl.cancel = cancel

	res, err := http.Head(dl.config.Url)
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode == http.StatusOK && res.Header.Get("Accept-Ranges") == "bytes" {
		contentSize, err := strconv.Atoi(res.Header.Get("Content-Length"))
		if err != nil {
			log.Fatal(err)
		}
		dl.multiDownload(contentSize)
	} else {
		dl.simpleDownload()
	}
}

// Server does not support partial download for this file
func (dl *downloader) simpleDownload() {
	if dl.config.Resume {
		log.Fatal("Cannot resume. Must be downloaded again")
	}

	// make a request
	res, err := http.Get(dl.config.Url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	// create the output file
	f, err := os.OpenFile(dl.config.OutFilename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	dl.progressBar = progressbar.DefaultBytes(int64(res.ContentLength), "downloading")

	// copy to output file
	buffer := make([]byte, dl.config.CopyBufferSize)
	_, err = io.CopyBuffer(io.MultiWriter(f, dl.progressBar), res.Body, buffer)
	if err != nil {
		log.Fatal(err)
	}
}

// download parallelly
func (dl *downloader) multiDownload(contentSize int) {
	partSize := contentSize / dl.config.Concurrency

	startRange := 0
	wg := &sync.WaitGroup{}
	wg.Add(dl.config.Concurrency)

	dl.progressBar = progressbar.DefaultBytes(int64(contentSize), "downloading")

	for i := 1; i <= dl.config.Concurrency; i++ {

		// handle resume
		downloaded := 0
		if dl.config.Resume {
			filePath := dl.getPartFilename(i)
			f, err := os.Open(filePath)
			if err == nil {
				fileInfo, err := f.Stat()
				if err == nil {
					downloaded = int(fileInfo.Size())
					// update progress bar
					dl.progressBar.Add64(int64(downloaded))
				}
			}
		}

		if i == dl.config.Concurrency {
			go dl.downloadPartial(startRange+downloaded, contentSize, i, wg)
		} else {
			go dl.downloadPartial(startRange+downloaded, startRange+partSize, i, wg)
		}

		startRange += partSize + 1
	}

	wg.Wait()
	if !dl.Paused {
		dl.merge()
	}
}

func (dl *downloader) merge() {
	destination, err := os.OpenFile(dl.config.OutFilename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer destination.Close()

	for i := 1; i <= dl.config.Concurrency; i++ {
		filename := dl.getPartFilename(i)
		source, err := os.OpenFile(filename, os.O_RDONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}
		io.Copy(destination, source)
		source.Close()
		os.Remove(filename)
	}
}

func (dl *downloader) downloadPartial(rangeStart, rangeStop int, partialNum int, wg *sync.WaitGroup) {
	defer wg.Done()
	if rangeStart >= rangeStop {
		// nothing to download
		return
	}

	// create a request
	req, err := http.NewRequest("GET", dl.config.Url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeStop))

	// make a request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	// create the output file
	outputPath := dl.getPartFilename(partialNum)
	flags := os.O_CREATE | os.O_WRONLY
	if dl.config.Resume {
		flags = flags | os.O_APPEND
	}
	f, err := os.OpenFile(outputPath, flags, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// copy to output file
	for {
		select {
		case <-dl.context.Done():
			return
		default:
			_, err = io.CopyN(io.MultiWriter(f, dl.progressBar), res.Body, int64(dl.config.CopyBufferSize))
			if err != nil {
				if err != io.EOF {
					log.Fatal(err)
				}
				return
			}
		}
	}
}
