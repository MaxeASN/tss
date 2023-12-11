package node

import (
	"crypto/ecdsa"
	"math/big"
	"os"
	"path"

	"github.com/bnb-chain/tss-lib/v2/crypto"
	"github.com/bnb-chain/tss-lib/v2/crypto/paillier"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/bnb-chain/tss/common"
	lib "github.com/ethereum/go-ethereum/crypto"
)

func loadSavedKeyForSign(config *common.TssConfig, sortedIds tss.SortedPartyIDs, signers map[string]int) keygen.LocalPartySaveData {
	result := loadSavedKey(config)
	filteredBigXj := make([]*crypto.ECPoint, 0)
	filteredPaillierPks := make([]*paillier.PublicKey, 0)
	filteredNTildej := make([]*big.Int, 0)
	filteredH1j := make([]*big.Int, 0)
	filteredH2j := make([]*big.Int, 0)
	filteredKs := make([]*big.Int, 0)
	for _, partyId := range sortedIds {
		keygenIdx := signers[partyId.Moniker]
		filteredBigXj = append(filteredBigXj, result.BigXj[keygenIdx])
		filteredPaillierPks = append(filteredPaillierPks, result.PaillierPKs[keygenIdx])
		filteredNTildej = append(filteredNTildej, result.NTildej[keygenIdx])
		filteredH1j = append(filteredH1j, result.H1j[keygenIdx])
		filteredH2j = append(filteredH2j, result.H2j[keygenIdx])
		filteredKs = append(filteredKs, result.Ks[keygenIdx])
	}
	filteredResult := keygen.LocalPartySaveData{
		LocalPreParams: keygen.LocalPreParams{
			PaillierSK: result.PaillierSK,
			NTildei:    result.NTildei,
			H1i:        result.H1i,
			H2i:        result.H2i,
		},
		LocalSecrets: keygen.LocalSecrets{
			Xi:      result.Xi,
			ShareID: result.ShareID,
		},
		Ks:          filteredKs,
		NTildej:     filteredNTildej,
		H1j:         filteredH1j,
		H2j:         filteredH2j,
		BigXj:       filteredBigXj,
		PaillierPKs: filteredPaillierPks,
		ECDSAPub:    result.ECDSAPub,
	}

	return filteredResult
}

func loadSavedKeyForRegroup(config *common.TssConfig, sortedIds tss.SortedPartyIDs, signers map[string]int) keygen.LocalPartySaveData {
	result := loadSavedKeyForSign(config, sortedIds, signers)

	if !config.IsOldCommittee {
		// TODO: negotiate with Luke to see how to fill non-loaded keys here
		for i := config.Parties; i < config.NewParties; i++ {
			result.BigXj = append(result.BigXj, result.BigXj[len(signers)-1])
			result.PaillierPKs = append(result.PaillierPKs, result.PaillierPKs[len(signers)-1])
			result.NTildej = append(result.NTildej, result.NTildej[len(signers)-1])
			result.H1j = append(result.H1j, result.H1j[len(signers)-1])
			result.H2j = append(result.H2j, result.H2j[len(signers)-1])
			result.Ks = append(result.Ks, result.Ks[len(signers)-1])
		}
	}
	return result
}

func loadSavedKey(config *common.TssConfig) keygen.LocalPartySaveData {
	wPriv, err := os.OpenFile(path.Join(config.Home, config.Vault, "sk.json"), os.O_RDONLY, 0400)
	if err != nil {
		common.Panic(err)
	}
	defer wPriv.Close()
	wPub, err := os.OpenFile(path.Join(config.Home, config.Vault, "pk.json"), os.O_RDONLY, 0400)
	if err != nil {
		common.Panic(err)
	}
	defer wPub.Close()

	result, _, err := common.Load(config.Password, wPriv, wPub) // TODO: validate nodeKey
	if err != nil {
		common.Panic(err)
	}
	return *result
}

func newEmptySaveData() keygen.LocalPartySaveData {
	return keygen.LocalPartySaveData{
		BigXj:       make([]*crypto.ECPoint, common.TssCfg.NewParties),
		PaillierPKs: make([]*paillier.PublicKey, common.TssCfg.NewParties),
		NTildej:     make([]*big.Int, common.TssCfg.NewParties),
		H1j:         make([]*big.Int, common.TssCfg.NewParties),
		H2j:         make([]*big.Int, common.TssCfg.NewParties),
	}
}

func appendIfNotExist(target []string, new string) []string {
	exist := false
	for _, old := range target {
		if old == new {
			exist = true
			break
		}
	}
	if !exist {
		target = append(target, new)
	}
	return target
}

func GetAddress(pubkey *ecdsa.PublicKey, prefix string) string {
	// generate ethereum address from ecdsa public key
	return lib.PubkeyToAddress(*pubkey).String()
}
