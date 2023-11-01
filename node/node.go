package node

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/bnb-chain/tss/client"
	"github.com/bnb-chain/tss/ssdp"
	"net"
	"os"
	"strings"
	"time"

	"github.com/bgentry/speakeasy"
	"github.com/bnb-chain/tss/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p"
	"github.com/multiformats/go-multiaddr"

	"google.golang.org/protobuf/proto"
)

type Node struct {
	TssCfg *common.TssConfig
	P2pCfg *common.P2PConfig

	bootstrapper *common.Bootstrapper
}

func New(cfg *common.TssConfig, isBootstraper bool) *Node {
	node := &Node{
		TssCfg: cfg,
	}
	// parse the bootstraper listing address
	src, err := common.ConvertMultiAddrStrToNormalAddr(common.TssCfg.ListenAddr)
	if err != nil {
		common.Panic(err)
	}
	// get listening addresses
	listenAddresses := getListenAddrs(common.TssCfg.ListenAddr)
	client.Logger.Debugf("This node is listening on: %v", listenAddresses)

	// channel id and password
	setChannelId()
	setChannelPasswd()

	// cal number of peers
	numOfPeers := common.TssCfg.Parties - 1

	// generate new bootstrapper
	// notice: bootstrapper is used to find the peers using ssdp service
	node.bootstrapper = common.NewBootstrapper(numOfPeers, cfg)

	// generate new ssdp protocol listener
	ssdpListener, err := net.Listen("tcp", src)
	if err != nil {
		common.Panic(err)
	}
	defer func() {
		err = ssdpListener.Close()
		if err != nil {
			common.Panic(err)
		}
		client.Logger.Info("closed ssdp listener")
	}()

	//done := make(chan bool)

	ctx, cancel := context.WithCancel(context.Background())
	// accept incoming connections
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				ssdpListener.Close()
				return
			default:
				client.Logger.Info("Waiting for incoming connections", "localAddr", ssdpListener.Addr().String())
				conn, err := ssdpListener.Accept()
				if err != nil {
					client.Logger.Error("ssdp listener accept error", "err", err)
					continue
				}

				// notice: this will store new peers
				handleConn(conn, node.bootstrapper)
			}
		}
	}(ctx)

	// peer discovery
	client.Logger.Info("Finding peer via ssdp, Searching ...", "listenAddresses", listenAddresses, "numOfPeers", numOfPeers)
	newPeerAddresses := findPeerAddrsViaSsdp(numOfPeers, listenAddresses)
	done := make(chan bool, 1)

	client.Logger.Debug("Find peer via ssdp ...", "peers", newPeerAddresses)

	go func() {
		for _, peer := range newPeerAddresses {
			go func(peer string) {
				dest, err := common.ConvertMultiAddrStrToNormalAddr(peer)
				if err != nil {
					common.Panic(err)
				}
				// trying to connect to the new peer
				conn, err := net.Dial("tcp", dest)
				if conn == nil || err != nil {
					common.Panic(err)
				}
				//defer conn.Close()
				handleConn(conn, node.bootstrapper)
			}(peer)
		}
		// check peer infos, check if we have the enough connected peers
		for {
			if node.bootstrapper.IsFinished() {
				cancel()
				done <- true
				break
			} else {
				time.Sleep(time.Second)
			}
		}
	}()

	// waiting here
	<-done

	node.bootstrapper.Peers.Range(func(id, value interface{}) bool {
		client.Logger.Debugf("Running New Peer", "address", value.(common.PeerInfo).RemoteAddr)
		return true
	})

	// update peer info to the bootstrapper
	updatePeerInfo(node.bootstrapper)

	return node
}

func (n *Node) Start(ctx context.Context) {
	//for {
	select {
	case <-ctx.Done():
		return
	default:
		c := client.NewTssClient(&common.TssCfg, client.KeygenMode, false)
		c.Start()
	}
	//}
}

func findPeerAddrsViaSsdp(peers int, addresses string) []string {
	ssdpSrv := ssdp.NewSsdpService(common.TssCfg.Moniker, common.TssCfg.Vault, addresses, peers, make(map[string]struct{}))
	ssdpSrv.CollectPeerAddrs()
	var collectedPeers []string
	ssdpSrv.PeerAddrs.Range(func(id, value interface{}) bool {
		if addr, ok := value.(string); ok {
			collectedPeers = append(collectedPeers, addr)
		}
		return true
	})
	return collectedPeers
}

func getListenAddrs(addr string) string {
	addrs, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		common.Panic(err)
	}
	hosts, err := libp2p.New(context.Background(), libp2p.ListenAddrs(addrs))
	if err != nil {
		common.Panic(err)
	}
	builder := strings.Builder{}
	for i, addr := range hosts.Addrs() {
		if i > 0 {
			fmt.Fprint(&builder, ", ")
		}
		fmt.Fprintf(&builder, "%s", addr.String())
	}
	hosts.Close()
	return builder.String()
}

func setChannelId() {
	if common.TssCfg.ChannelId != "" {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	channelId, err := common.GetString("please set channel id of this session", reader)
	if err != nil {
		common.Panic(err)
	}
	if len(channelId) != 11 {
		common.Panic(fmt.Errorf("channelId format is invalid"))
	}
	common.TssCfg.ChannelId = channelId
}

func setChannelPasswd() {
	if pw := common.TssCfg.ChannelPassword; pw != "" {
		return
	}

	if p, err := speakeasy.Ask("> please input password (AGREED offline with peers) of this session:"); err == nil {
		if p == "" {
			common.Panic(fmt.Errorf("channel password should not be empty"))
		}
		common.TssCfg.ChannelPassword = p
	} else {
		common.Panic(err)
	}
}

func handleConn(conn net.Conn, bootsraper *common.Bootstrapper) {
	log.Debug("handle new connection", "remote", conn.RemoteAddr().String())

	// send bootstrap message
	sendBootstrapMessage(conn, bootsraper.Msg)
	// read bootstrap message
	readBootstrapMessage(conn, bootsraper)
}

func sendBootstrapMessage(conn net.Conn, msg *common.BootstrapMessage) {

	realIP := strings.SplitN(conn.LocalAddr().String(), ":", 2)[0]
	connMsg := &common.BootstrapMessage{
		ChannelId: msg.ChannelId,
		PeerInfo:  msg.PeerInfo,
		Addr:      common.ReplaceIpInAddr(msg.Addr, realIP),
	}
	payload, err := proto.Marshal(connMsg)
	if err != nil {
		client.Logger.Error("marshal bootstrap message error", "err", err)
		return
	}

	messageLength := int32(len(payload))
	err = binary.Write(conn, binary.BigEndian, &messageLength)
	if err != nil {
		common.SkipTcpClosePanic(fmt.Errorf("failed to write bootstrap message length: %v", err))
		return
	}

	n, err := conn.Write(payload)
	if n != len(payload) {
		client.Logger.Error("write bootstrap message error", "err", err)
		return
	}
	return
}

func readBootstrapMessage(conn net.Conn, bootstrapper *common.Bootstrapper) {
	var messageLength int32
	err := binary.Read(conn, binary.BigEndian, &messageLength)
	if err != nil {
		common.SkipTcpClosePanic(fmt.Errorf("failed to read bootstrap message length: %v", err))
		return
	}

	//log.Info("!!!!!!!!!!!!! READ bootstrap message length !!!!!!!!!!!!!", "messageLength", messageLength)

	payload := make([]byte, messageLength)
	n, err := conn.Read(payload)
	//n, err := io.ReadFull(conn, payload)
	if int32(n) != messageLength {
		client.Logger.Error("read bootstrap message length error", "err", err, "read", n, "length", messageLength, "from", conn.RemoteAddr().String())
	}
	if err != nil {
		client.Logger.Error("read bootstrap message error", "err", err, "from", conn.RemoteAddr().String())
		return
	}
	// unmarshal bootstrap message
	var peerMsg common.BootstrapMessage
	err = proto.Unmarshal(payload, &peerMsg)
	if err != nil {
		client.Logger.Error("unmarshal bootstrap message error", "err", err, "payload", payload)
		return
	}

	if err = bootstrapper.HandleBootstrapMsg(peerMsg); err != nil {
		client.Logger.Error("handle bootstrap message error", "err", err)
	}
	client.Logger.Info("Read bootstrap message successfully", "remote", conn.RemoteAddr().String())
}

func updatePeerInfo(bootstrapper *common.Bootstrapper) {
	peerAddrs := make([]string, 0)
	expectedPeerAddrs := make([]string, 0)

	newPeerAddrs := make([]string, 0)
	newExpectedPeerAddrs := make([]string, 0)

	bootstrapper.Peers.Range(func(id, value any) bool {
		if p, ok := value.(common.PeerInfo); ok {
			client.Logger.Debugf("Trying to update peer info", "address", p.RemoteAddr)
			// if not regroup mode
			if common.TssCfg.BMode != common.PreRegroupMode {
				peerAddrs = append(peerAddrs, p.RemoteAddr)
				expectedPeerAddrs = append(expectedPeerAddrs, fmt.Sprintf("%s@%s", p.Moniker, p.Id))
				// update common.TssCfg
				common.TssCfg.PeerAddrs, common.TssCfg.ExpectedPeers = mergePeerInfo(
					common.TssCfg.PeerAddrs,
					common.TssCfg.ExpectedPeers,
					peerAddrs,
					expectedPeerAddrs,
				)

			} else {
				newPeerAddrs = append(newPeerAddrs, p.RemoteAddr)
				newExpectedPeerAddrs = append(newExpectedPeerAddrs, fmt.Sprintf("%s@%s", p.Moniker, p.RemoteAddr))
				// update common.TssCfg
				common.TssCfg.NewPeerAddrs, common.TssCfg.ExpectedNewPeers = mergePeerInfo(
					common.TssCfg.NewPeerAddrs,
					common.TssCfg.ExpectedNewPeers,
					newPeerAddrs,
					newExpectedPeerAddrs,
				)
			}
			return true
		} else {
			client.Logger.Errorf("Failed to update peer info, address: %s", p.RemoteAddr, "Peer info is not of type %T", value)
			return false
		}
	})

	// log
	client.Logger.Debugf("Bootstrapper Peer Length: %d. Expected Peer Length: %d", len(peerAddrs), len(expectedPeerAddrs))
}

func mergePeerInfo(currentPeerAddrs, currentExpectedPeers, newPeerAddrs, newExpectedPeers []string) ([]string, []string) {
	needMerge := make(map[string]string) // map of expected peer addrs -> peer addrs
	//
	for i, p := range currentExpectedPeers {
		needMerge[p] = currentPeerAddrs[i]
	}
	for i, p := range newExpectedPeers {
		needMerge[p] = newPeerAddrs[i]
	}
	// cache
	updatedPeers := make([]string, 0)
	updatedPeerAddrs := make([]string, 0)
	// merge
	for p, addr := range needMerge {
		updatedPeers = append(updatedPeers, p)
		updatedPeerAddrs = append(updatedPeerAddrs, addr)
	}
	return updatedPeerAddrs, updatedPeers
}
