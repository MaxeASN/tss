package main

import (
	"context"
	"errors"
	"github.com/bnb-chain/tss/node"
	"github.com/bnb-chain/tss/worker"
	"github.com/knadh/koanf/providers/posflag"

	"github.com/bnb-chain/tss/api"
	"github.com/knadh/koanf"
	flag "github.com/spf13/pflag"
)

func ConfigAddOptions(f *flag.FlagSet) {
	api.APIConfigAddOptions("api", f)
	worker.WorkerConfigAddOptions("worker", f)
	node.NodeCliConfigAddOptions("node", f)
}

func ParseTssConfig(ctx context.Context, args []string) (*api.APIConfig, *worker.WorkerConfig, *node.NodeCliConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	ConfigAddOptions(f)

	// parse args
	if err := f.Parse(args); err != nil {
		return nil, nil, nil, err
	}

	if f.NArg() != 0 {
		return nil, nil, nil, errors.New("unexpected number of arguments")
	}

	// generate koanf config
	var k = koanf.New(".")
	if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		return nil, nil, nil, errors.New("failed to parse config from command line")
	}

	// generate api config
	apiConfig := &api.APIConfig{
		Enable:   k.Bool("api.enable"),
		Endpoint: k.String("api.endpoint"),
		API:      k.Strings("api.allowed-apis"),
	}

	// generate worker config
	workerConfig := &worker.WorkerConfig{
		WorkerLimit: k.Int("worker.limit"),
	}

	// generate node config
	nodeCliConfig := &node.NodeCliConfig{
		ChannelId:  k.String("node.channel_id"),
		ChannelPwd: k.String("node.channel_pwd"),
		SSDPAddr:   k.String("node.ssdp_addr"),
		PeerId:     k.String("node.peer_id"),
		Moniker:    k.String("node.moniker"),
		Vault:      k.String("node.vault"),
		BMode:      k.Int("node.bmode"),
		Parties:    k.Int("node.parties"),
		Threshold:  k.Int("node.threshold"),
	}

	return apiConfig, workerConfig, nodeCliConfig, nil

}
