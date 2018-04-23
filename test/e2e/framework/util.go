package framework

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
)

func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

func log(level string, format string, args ...interface{}) {
	fmt.Fprintf(ginkgo.GinkgoWriter, nowStamp()+": "+level+": "+format+"\n", args...)
}

// Logf logs in e2e framework
func Logf(format string, args ...interface{}) {
	log("INFO", format, args...)
}

// Failf reports a failure in the current e2e
func Failf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log("INFO", msg)
	ginkgo.Fail(nowStamp()+": "+msg, 1)
}

// LogAndReturnErrorf logs and return an error
func LogAndReturnErrorf(format string, args ...interface{}) error {
	Logf(format, args...)
	return fmt.Errorf(format, args...)
}
