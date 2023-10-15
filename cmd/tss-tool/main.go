package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/bnb-chain/tss/cmd/tss-tool/commands"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6062", nil))
	}()

	commands.Execute()
}
