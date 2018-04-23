package main

import (
	"context"
	goflag "flag"
	"os"
	"runtime"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/amadeusitgroup/redis-operator/pkg/operator"
	"github.com/amadeusitgroup/redis-operator/pkg/signal"
	"github.com/amadeusitgroup/redis-operator/pkg/utils"
)

func main() {
	utils.BuildInfos()
	runtime.GOMAXPROCS(runtime.NumCPU())

	config := operator.NewRedisOperatorConfig()
	config.AddFlags(pflag.CommandLine)

	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	goflag.CommandLine.Parse([]string{})

	op := operator.NewRedisOperator(config)

	if err := run(op); err != nil {
		glog.Errorf("RedisOperator returns an error:%v", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run(op *operator.RedisOperator) error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	go signal.HandleSignal(cancelFunc)

	op.Run(ctx.Done())

	return nil
}
