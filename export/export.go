package export

// Exporter interface is for extracting data to external resources.
// Geziyor calls every extractors Export functions before any scraping starts.
// Export functions should wait for new data from exports chan.
type Exporter interface {
	Export(exports chan interface{})
}
