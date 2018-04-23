package process

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
)

// Interface interface for process
type Interface interface {
	Init() error
	Clear()
	Start(stop <-chan struct{}) error
}

// ExecProcess execute a process
func ExecProcess(p Interface) error {
	defer glog.Flush()
	err := p.Init()
	defer p.Clear()

	if err != nil {
		glog.Errorf("Process cannot be initialized, Stopping. error: %v", err)
		return err
	}

	glog.Info("Starting...")
	sigs := make(chan os.Signal, 1)
	stop := make(chan struct{}, 1)

	signal.Notify(sigs, syscall.SIGINT)
	signal.Notify(sigs, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		glog.Info("Signal: ", sig)
		close(stop)
	}()

	return p.Start(stop)
}
