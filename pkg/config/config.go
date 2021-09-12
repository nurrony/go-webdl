package config

// Download Configuration
type Config struct {

	// Url to download from
	Url string

	// number of concurrent download process
	Concurrency int

	// output filename can have absolute directory path
	OutFilename    string

	// The buffer size in KiB to copy from response body
	CopyBufferSize int

	// is in resume mode?
	Resume bool
}
