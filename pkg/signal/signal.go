package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
)

// HandleSignal used to listen several os signal and then execute the cancel function
func HandleSignal(cancelFunc context.CancelFunc) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	sig := <-sigc
	glog.Infof("Signal received: %s, stop the process", sig.String())
	cancelFunc()
}
