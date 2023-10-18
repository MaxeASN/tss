package main

import (
	"context"
	"fmt"
	"github.com/bnb-chain/tss/api"
	"github.com/bnb-chain/tss/worker"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"log"
	"os"
	"os/signal"
)

var rpcAPI = []rpc.API{}

func main() {
	os.Exit(mainImpl())
}

func mainImpl() int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	apiConfig, workerConfig, err := ParseTssConfig(ctx, os.Args[1:])
	if err != nil {
		return 1
	}

	api := api.NewApiService(apiConfig)
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
			log.Fatalf("failed to start http server: %s, error=%v", addr, err)
		}
		fmt.Printf("MPC API Service Started, endpoint: %s\n", fmt.Sprintf("http://%v/", addr))
		defer func() {
			log.Print("Api Service stopping...")
			s.Shutdown(ctx)
		}()
	}

	// new worker with config
	newWorker := worker.NewWorker(ctx, workerConfig)
	newWorker.Start()
	//go func() {
	abortChan := make(chan os.Signal, 1)
	signal.Notify(abortChan, os.Interrupt)
	sig := <-abortChan
	log.Printf("\n----------------  Got signal: %v  ---------------- \n", sig)
	// stop the worker service
	{
		newWorker.Stop()
	}
	return 0
}
