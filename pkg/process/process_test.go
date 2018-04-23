package process

import (
	"errors"
	"syscall"
	"testing"
)

type TestProcess struct {
	Success   bool
	IsInit    bool
	IsStarted bool
	IsCleared bool
	IsStopped bool
	Synchro   chan bool
}

func NewTestProcess(synchro chan bool, success bool) Interface {
	return &TestProcess{Success: success, IsInit: false, IsStarted: false, IsStopped: false, Synchro: synchro}
}

func (tp *TestProcess) Init() error {
	if tp.Success {
		tp.IsInit = true
		return nil
	}

	return errors.New("error")
}

func (tp *TestProcess) Clear() {
	tp.IsCleared = true
}

func (tp *TestProcess) Start(stop <-chan struct{}) error {
	tp.IsStarted = true
	tp.Synchro <- true
	<-stop
	tp.IsStopped = true
	return nil
}

func TestExecProcess(t *testing.T) {
	synchro := make(chan bool, 1)
	tp := NewTestProcess(synchro, true)

	go func() {
		<-synchro
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	err := ExecProcess(tp)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	ttp := tp.(*TestProcess)
	if !ttp.IsInit || !ttp.IsStarted || !ttp.IsCleared || !ttp.IsStopped {
		t.Errorf("Error One state missing: %#v", ttp)
	}
}

func TestFailExecProcess(t *testing.T) {
	synchro := make(chan bool, 1)
	tp := NewTestProcess(synchro, false)

	err := ExecProcess(tp)
	if err == nil {
		t.Error("Should have returned an error")
	}
}
