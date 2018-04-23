package main

import (
	"context"
	goflag "flag"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/amadeusitgroup/redis-operator/pkg/redisnode"
	"github.com/amadeusitgroup/redis-operator/pkg/signal"
	"github.com/amadeusitgroup/redis-operator/pkg/utils"
)

func main() {
	utils.BuildInfos()
	config := redisnode.NewRedisNodeConfig()
	config.AddFlags(pflag.CommandLine)

	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	goflag.CommandLine.Parse([]string{})

	rn := redisnode.NewRedisNode(config)

	if err := run(rn); err != nil {
		glog.Errorf("RedisNode returns an error:%v", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run(rn *redisnode.RedisNode) error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	go signal.HandleSignal(cancelFunc)

	rn.Run(ctx.Done())

	return nil
}
