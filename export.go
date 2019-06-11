package geziyor

// Exporter interface is for extracting data to external resources
type Exporter interface {
	Export(exports chan interface{})
}
