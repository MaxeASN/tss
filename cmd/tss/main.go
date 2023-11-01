package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"

	"github.com/bnb-chain/tss/api"
	"github.com/bnb-chain/tss/client"
	"github.com/bnb-chain/tss/common"
	tssNode "github.com/bnb-chain/tss/node"
	"github.com/bnb-chain/tss/worker"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ipfs/go-log"
)

var rpcAPI = []rpc.API{}
var logger = log.Logger("main")

func main() {
	os.Exit(mainImpl())
}

func mainImpl() int {

	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	apiConfig, workerConfig, nodeCliConfig, err := ParseTssConfig(ctx, os.Args[1:])
	if err != nil {
		return 1
	}

	// new worker with config
	newWorker := worker.NewWorker(ctx, workerConfig)
	newWorker.Start()

	api := api.NewApiService(apiConfig, newWorker)
	apiService := rpc.API{
		Namespace: "mpc",
		Version:   "1.0",
		Service:   api,
		Public:    true,
	}

	rpcAPI = append(rpcAPI, apiService)

	// if api is enabled, it starts an mpc node service with an api service,
	// if not, it can only start a normal mpc node service.
	if apiConfig.Enable {
		// new node server
		srv := rpc.NewServer()
		err = node.RegisterApis(rpcAPI, []string{"mpc"}, srv)
		if err != nil {
			return 1
		}

		handler := node.NewHTTPHandlerStack(srv, []string{}, []string{}, nil)
		s, addr, err := node.StartHTTPEndpoint(apiConfig.Endpoint, rpc.DefaultHTTPTimeouts, handler)
		if err != nil {
			client.Logger.Errorf("failed to start http server: %s, error=%v", addr, err)
		}
		client.Logger.Infof("MPC API Service Started, endpoint: %s\n", fmt.Sprintf("http://%v/", addr))
		defer func() {
			client.Logger.Warning("Api Service stopping...")
			s.Shutdown(ctx)
		}()
	}

	//go func() {
	go func() {
		tssNode.ParseNodeConfig(nodeCliConfig)
		initLogLevel(common.TssCfg)

		node := tssNode.New(&common.TssCfg, true)
		node.Start(ctx)
	}()

	abortChan := make(chan os.Signal, 1)
	signal.Notify(abortChan, os.Interrupt)
	sig := <-abortChan

	logger.Infof("----------------  Got signal: %v  ----------------", sig)
	// stop the worker service
	{
		cancelFunc()
		newWorker.Stop()
	}
	return 0
}

func initLogLevel(cfg common.TssConfig) {
	log.SetLogLevel("main", "debug")
	log.SetLogLevel("tss", cfg.LogLevel)
	log.SetLogLevel("tss-lib", cfg.LogLevel)
	log.SetLogLevel("srv", cfg.LogLevel)
	log.SetLogLevel("trans", cfg.LogLevel)
	log.SetLogLevel("p2p_utils", cfg.LogLevel)
	log.SetLogLevel("common", cfg.LogLevel)

	// libp2p loggers
	log.SetLogLevel("dht", "error")
	log.SetLogLevel("discovery", "error")
	log.SetLogLevel("swarm2", "error")
	log.SetLogLevel("stream-upgrader", "error")
}
