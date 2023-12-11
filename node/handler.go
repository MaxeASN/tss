package node

import (
	"context"
	"fmt"
	lib "github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/bnb-chain/tss/common"
	"github.com/bnb-chain/tss/p2p"
	"github.com/bnb-chain/tss/task"
	"google.golang.org/protobuf/reflect/protoreflect"
	"math/big"
	"time"
)

type Handler struct {
	params        *tss.Parameters
	preParams     *keygen.LocalPreParams
	regroupParams *tss.ReSharingParameters

	localParty  tss.Party
	transporter common.Transporter

	mode      common.OpType
	key       *keygen.LocalPartySaveData
	signature []byte

	saveCh  chan keygen.LocalPartySaveData
	signCh  chan lib.SignatureData
	sendCh  chan tss.Message
	errorCh chan error
}

var localPreParams *keygen.LocalPreParams
var localParties, localThreshold int

// GenerateLocalPreParams generates the local pre-shared parameters
func GenerateLocalPreParams(parties, threshold int) {
	params, _ := keygen.GeneratePreParams(time.Minute * 10)
	localPreParams = params
	localParties = parties
	localThreshold = threshold
	return
}

// prepare handler
func Prepare(ctx context.Context) task.PrepareHandler {
	return func(ctx context.Context, event *task.TaskEvent) (result bool) {
		// prepare for generate localparty
		done := make(chan struct{})

		go func(done chan struct{}) {
			//result = make(chan bool)
			select {
			case <-ctx.Done():
				result = false
			default:
				//for {
				//
				//}

				result = true
				done <- struct{}{}
			}
		}(done)

		<-done
		return result
	}
}

func newHandler(preParams *keygen.LocalPreParams, t common.Transporter) *Handler {
	// new handler
	var h = &Handler{
		params:        nil,
		preParams:     preParams,
		regroupParams: &tss.ReSharingParameters{},
		//mode:          mode,
		transporter: t,
		saveCh:      make(chan keygen.LocalPartySaveData),
		signCh:      make(chan lib.SignatureData),
		sendCh:      make(chan tss.Message, 10*2),
		errorCh:     make(chan error),
	}
	return h
}

func NewHandlerFactory() task.HandlerFactory {
	return func(ctx context.Context) task.IEventHandler {
		select {
		case <-ctx.Done():
			return nil
		default:
			//todo uncomment when factory mode
			//if localPreParams == nil {
			//	return nil
			//}
			return newHandler(localPreParams, localTransporter)
		}
	}
}

func (h *Handler) Preprocess(event *task.TaskEvent) bool {

	// transporter is not ready
	if h.transporter == nil {
		h.transporter = localTransporter
	}
	// set local party running mode
	h.mode = event.Op

	// init `localParty` depending on the mode
	var sortedIds tss.SortedPartyIDs
	p2pCtx := tss.NewPeerContext(sortedIds)

	var partyId = tss.NewPartyID(
		string(common.TssCfg.Id),
		common.TssCfg.Moniker,
		new(big.Int).SetBytes(lib.SHA512_256([]byte(common.TssCfg.Id))))

	var localParty tss.Party
	if h.mode == common.KeyGenType { // keygen type
		params := tss.NewParameters(tss.EC(), p2pCtx, partyId, 3, 1)
		localParty = keygen.NewLocalParty(params, h.sendCh, h.saveCh)
		h.localParty = localParty
		// todo
	} else if h.mode == common.SignType {
		key := loadSavedKeyForSign(&common.TssCfg, sortedIds, nil)
		fmt.Printf("address %s is ready to sign...\n", GetAddress(key.ECDSAPub.ToECDSAPubKey(), ""))
		params := tss.NewParameters(tss.EC(), p2pCtx, partyId, 3, 1)
		// message to be signed
		message, ok := big.NewInt(0).SetString(event.Message, 16)
		if !ok {
			return false
		}
		// local party
		localParty := signing.NewLocalParty(message, params, key, h.sendCh, h.signCh)

		//
		h.key = &key
		h.params = params
		h.localParty = localParty

	} else if h.mode == common.RegroupType {

	}


	return true
}

type TestMessage struct {
	Message []byte
}

func (m *TestMessage) ProtoReflect() protoreflect.Message {
	//TODO implement me
	panic("implement me")
}

func (m *TestMessage) ValidateBasic() bool {
	return m.Message != nil
}

func (h *Handler) Start() {
}

func (h *Handler) Result() <-chan any {
	var result = make(chan any)
	go func() {
		select {
		case saveData := <-h.saveCh:
			result <- saveData
		case signature := <-h.signCh:
			result <- signature
		}
	}()

	return result
}

func (h *Handler) Error() <-chan error {
	return h.errorCh
}

