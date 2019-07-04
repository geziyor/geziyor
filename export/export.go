package export

// Exporter interface is for extracting data to external resources.
// Export functions should wait for new data from exports chan.
type Exporter interface {
	Export(exports chan interface{})
}
