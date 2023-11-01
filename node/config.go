package node

import (
	"bufio"
	"github.com/bnb-chain/tss/common"
	flag "github.com/spf13/pflag"
	"os"
)

type NodeCliConfig struct {
	ChannelId  string `koanf:"channel_id"`
	ChannelPwd string `koanf:"channel_pwd"`

	SSDPAddr string `koanf:"ssdp_addr"`
	PeerId   string `koanf:"peer_id"`

	Moniker string `koanf:"moniker"`
	Vault   string `koanf:"vault"`
	BMode   int    `koanf:"bmode"`

	Parties   int `koanf:"parties"`
	Threshold int `koanf:"threshold"`

	LogLevel int `koanf:"log_level"`
}

var DefaultNodeCliConfig = NodeCliConfig{
	ChannelId:  "7236647E25C",
	ChannelPwd: "thisIsPassword",
	SSDPAddr:   "",
	PeerId:     "",
	Moniker:    "",
	Vault:      "default",
	BMode:      0,
	Parties:    3,
	Threshold:  2,
	LogLevel:   1,
}

func CliConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".channel_id", DefaultNodeCliConfig.ChannelId, "channel id of p2p connection")
	f.String(prefix+".channel_pwd", DefaultNodeCliConfig.ChannelPwd, "channel password of p2p connection")
	f.String(prefix+".ssdp_addr", DefaultNodeCliConfig.SSDPAddr, "ssdp discover address")
	f.String(prefix+".peer_id", DefaultNodeCliConfig.PeerId, "p2p peer id")
	f.String(prefix+".moniker", DefaultNodeCliConfig.Moniker, "moniker of current party")
	f.String(prefix+".vault", DefaultNodeCliConfig.Vault, "name of vault of this party")
	f.Int(prefix+".bmode", DefaultNodeCliConfig.BMode, "bit mode of current worker") // todo remove this flag to the worker
	f.Int(prefix+".parties", DefaultNodeCliConfig.Parties, "total parties")
	f.Int(prefix+".threshold", DefaultNodeCliConfig.Threshold, "threshold of this party")
	f.Int(prefix+"log_level", DefaultNodeCliConfig.LogLevel, "set log level of current worker")
}

func ParseNodeConfig(cfg *NodeCliConfig) {

	common.TssCfg.ChannelId = cfg.ChannelId
	common.TssCfg.ChannelPassword = cfg.ChannelPwd
	common.TssCfg.Vault = cfg.Vault
	common.TssCfg.BMode = common.BootstrapMode(cfg.BMode)
	common.TssCfg.Parties = cfg.Parties
	common.TssCfg.Threshold = cfg.Threshold

	reader := bufio.NewReader(os.Stdin)

	// set listening address
	if cfg.SSDPAddr == "" {
		address, err := common.GetString("Pls input listen address: ", reader)
		if err != nil {
			panic(err)
		}
		common.TssCfg.ListenAddr = address
	} else {
		common.TssCfg.ListenAddr = cfg.SSDPAddr
	}

	// set peer id
	if cfg.PeerId == "" {
		id, err := common.GetString("Pls input Peer id: ", reader)
		if err != nil {
			panic(err)
		}
		common.TssCfg.Id = common.TssClientId(id)
	} else {
		common.TssCfg.Id = common.TssClientId(cfg.PeerId)
	}

	// set moniker
	if cfg.Moniker == "" {
		moniker, err := common.GetString("Pls input moniker: ", reader)
		if err != nil {
			panic(err)
		}
		common.TssCfg.Moniker = moniker
	} else {
		common.TssCfg.Moniker = cfg.Moniker
	}

	// set log level
	common.TssCfg.LogLevel = logLevel(cfg.LogLevel)

	//
	common.TssCfg.Id = common.TssClientId(cfg.PeerId)

	//
	common.TssCfg.Home = cfg.Moniker

	tssCfg, err := common.LoadConfig(cfg.Moniker, cfg.Vault, "123456789")
	if err != nil {
		panic(err)
	}
	common.TssCfg.Home = tssCfg.Home
	common.TssCfg.ProfileAddr = tssCfg.ProfileAddr
	common.TssCfg.KDFConfig = tssCfg.KDFConfig

}

func logLevel(level int) string {
	switch level {
	case 0:
		return "debug"
	case 1:
		return "info"
	case 2:
		return "warn"
	case 3:
		return "error"
	default:
		return "info"
	}
}
