package utils

// LogWriter struct representation of a log writter, containing the pointer of the logging function
type LogWriter struct {
	logFunc func(args ...interface{})
}

// NewLogWriter create a new LogWriter with the pointer of the logging function in input
func NewLogWriter(f func(args ...interface{})) *LogWriter {
	w := &LogWriter{logFunc: f}
	return w
}

// Write implements the standard Write interface: it writes using the logging function
func (w LogWriter) Write(p []byte) (n int, err error) {
	w.logFunc(string(p))
	return len(p), nil
}
