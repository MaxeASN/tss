package node

import (
	"bufio"
	"context"
	"fmt"
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
	log.Info("Node listening addresses", "addresses", listenAddresses)

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
	}()
	ctx, cancel := context.WithCancel(context.Background())
	// accept incoming connections
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				log.Info("Waiting for incoming connections...")
				conn, err := ssdpListener.Accept()
				if err != nil {
					log.Error("ssdp listener accept error", "err", err)
					continue
				}
				log.Info("ssdp listener accept new connection", "remote", conn.RemoteAddr().String())

				// notice: this will store new peers
				handleConn(conn, node.bootstrapper)
			}
		}
	}(ctx)

	// peer discovery
	log.Info("Finding peer via ssdp, Searching ...", "listenAddresses", listenAddresses, "numOfPeers", numOfPeers)
	newPeerAddresses := findPeerAddrsViaSsdp(numOfPeers, listenAddresses)
	done := make(chan bool, 1)

	log.Info("Find peer via ssdp ...", "peers", newPeerAddresses)

	go func() {
		for _, peer := range newPeerAddresses {
			log.Info("Found new Node Peer address", "address", peer)
			go func(peer string) {
				dest, err := common.ConvertMultiAddrStrToNormalAddr(peer)
				if err != nil {
					common.Panic(err)
				}
				// trying to connect to the new peer
				log.Debug("Trying to connect to Node", "remote", dest)
				conn, err := net.Dial("tcp", dest)
				if conn == nil || err != nil {
					common.Panic(err)
				}
				//
				log.Info("Connected to Node", "remote", dest)
				//
				handleConn(conn, node.bootsraper)
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

	return node
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
	for _, peer := range collectedPeers {
		fmt.Println(">>>>>>>>>>>>>>> ", peer)
	}
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
	n := sendBootstrapMessage(conn, bootsraper.Msg)
	// read bootstrap message
	readBootstrapMessage(conn, bootsraper, n)
}

func sendBootstrapMessage(conn net.Conn, msg *common.BootstrapMessage) int {
	realIP := strings.SplitN(conn.LocalAddr().String(), ":", 2)[0]
	connMsg := &common.BootstrapMessage{
		ChannelId: common.TssCfg.ChannelId,
		PeerInfo:  msg.PeerInfo,
		Addr:      common.ReplaceIpInAddr(msg.Addr, realIP),
	}
	payload, err := proto.Marshal(connMsg)
	if err != nil {
		log.Error("marshal bootstrap message error", "err", err)
		return 0
	}

	n, err := conn.Write(payload)
	if n != len(payload) {
		log.Error("write bootstrap message error", "err", err)
		return 0
	}

	// write bootstrap message successfully
	log.Info(">>>>>>> write bytes", "payload", payload)
	log.Info("Write bootstrap message successfully", "remote", conn.RemoteAddr().String(), "length", n)
	return n
}

func readBootstrapMessage(conn net.Conn, bootstraper *common.Bootstrapper, l int) {
	payload := make([]byte, l)
	n, err := conn.Read(payload)
	if err != nil || n != l {
		log.Error("read bootstrap message error", "err", err)
		return
	}

	// unmarshal bootstrap message
	var peerMsg common.BootstrapMessage
	err = proto.Unmarshal(payload, &peerMsg)
	if err != nil {
		log.Error("unmarshal bootstrap message error", "err", err)
		return
	}

		log.Error("handle bootstrap message error", "err", err)
	if err = bootstrapper.HandleBootstrapMsg(peerMsg); err != nil {
	}
	log.Info("Read bootstrap message successfully", "remote", conn.RemoteAddr().String())
}
