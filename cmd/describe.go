package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

func init() {
	rootCmd.AddCommand(describeCmd)
}

// fmt.Printf is deliberately used in this command
var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "show config and address of a tss vault",
	Long:  "",
	PreRun: func(cmd *cobra.Command, args []string) {
		passphrase := askPassphrase()
		vault := askVault()
		if err := common.ReadConfigFromHome(viper.GetViper(), viper.GetString(flagHome), vault, passphrase); err != nil {
			panic(err)
		}
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		pubKey, err := common.LoadEcdsaPubkey(viper.GetString(flagHome), viper.GetString(flagVault), common.TssCfg.Password)
		if err != nil {
			fmt.Printf("cannot load public key, maybe not keygen yet: %v", err)
		}
		if pubKey != nil {
			addr, err := client.GetAddress(*pubKey, viper.GetString(flagPrefix))
			if err != nil {
				panic(err)
			}
			fmt.Printf("address of this vault: %s\n", addr)
		}
		cfg, err := json.MarshalIndent(common.TssCfg, "", "\t")
		if err != nil {
			panic(err)
		}
		fmt.Printf("config of this vault:\n%s\n", string(cfg))
	},
}
