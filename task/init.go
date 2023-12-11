package task

import (
	"bufio"
	"github.com/bnb-chain/tss/client"
	"github.com/bnb-chain/tss/common"
	"github.com/spf13/viper"
	"os"
	"path"
)

func checkOverride() {
	if _, err := os.Stat(path.Join(common.TssCfg.Home, common.TssCfg.Vault, "sk.json")); err == nil {
		// we have already done keygen before
		reader := bufio.NewReader(os.Stdin)
		answer, err := common.GetBool("Vault already generated, do you like override it[y/N]: ", false, reader)
		if err != nil {
			common.Panic(err)
		}
		if !answer {
			client.Logger.Info("nothing happened")
			os.Exit(0)
		} else {
			common.TssCfg.Parties = viper.GetInt("parties")
			common.TssCfg.Threshold = viper.GetInt("threshold")
		}
	}
}
