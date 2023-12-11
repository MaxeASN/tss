package common

type TssClientId string // Pretty() of peer.ID

func (cid *TssClientId) String() string {
	return string(*cid)
}

func (cl *TssClientId) Set(value string) error {
	*cl = TssClientId(value)
	return nil
}

type P2pMessageWrapper struct {
	EventId             string // each message has a unique event id
	MessageWrapperBytes []byte // marshaled protobuf message
}

type OpType int8

const (
	KeyGenType OpType = iota
	SignType
	RegroupType
	GenChannelIdType
	InitType
)

func (t OpType) String() string {
	switch t {
	case KeyGenType:
		return "keygen"
	case SignType:
		return "sign"
	case RegroupType:
		return "regroup"
	default:
		logger.Error("Unknown operation type")
		return ""
	}
}
