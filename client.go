package litrpcclient

import (
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"strings"

	"github.com/mit-dci/lit/litrpc"
	"golang.org/x/net/websocket"
)

type LitRpcClient struct {
	wsConn          *websocket.Conn
	rpcConn         *rpc.Client
	listeningStatus uint8
}

// NewClient creates a new LitRpcClient and connects to the given
// hostname and port
func NewClient(host string, port int32) (*LitRpcClient, error) {
	client := new(LitRpcClient)
	var err error
	client.wsConn, err = websocket.Dial(fmt.Sprintf("ws://%s:%d/ws", host, port), "", "http://127.0.0.1/")
	if err != nil {
		return nil, err
	}
	client.rpcConn = jsonrpc.NewClient(client.wsConn)
	return client, nil
}

// Close Disconnects from the LIT node
func (c *LitRpcClient) Close() {
	c.wsConn.Close()
}

//Listen instructs LIT to listen for incoming connections. By default, LIT will not
//listen. If LIT was already listening for incoming connections, this method
//will just resolve.
func (c *LitRpcClient) Listen(port string) error {
	args := new(litrpc.ListenArgs)
	args.Port = port

	reply := new(litrpc.ListeningPortsReply)
	err := c.rpcConn.Call("LitRPC.Listen", args, reply)
	if err != nil {
		if strings.Index(err.Error(), "already in use") == -1 {
			return err
		}
	}
	c.listeningStatus = 1
	return nil
}

// IsListening checks if LIT is currently listening on any port.
func (c *LitRpcClient) IsListening() (bool, error) {
	if c.listeningStatus > 0 {
		return (c.listeningStatus == 1), nil
	}

	args := new(litrpc.NoArgs)
	reply := new(litrpc.ListeningPortsReply)
	err := c.rpcConn.Call("LitRPC.GetListeningPorts", args, reply)
	if err != nil {
		return false, err
	}
	c.listeningStatus = 1
	if reply.LisIpPorts == nil {
		c.listeningStatus = 2
	}
	return (c.listeningStatus == 1), nil
}

// GetLNAddress returns the LN address for this node
func (c *LitRpcClient) GetLNAddress() (string, error) {
	args := new(litrpc.NoArgs)

	reply := new(litrpc.ListeningPortsReply)
	err := c.rpcConn.Call("LitRPC.GetListeningPorts", args, reply)
	if err != nil {
		return "", err
	}
	return reply.Adr, nil
}

// Connect connects to another LIT node. address is mandatory, host and port can be left empty / 0.
func (c *LitRpcClient) Connect(address, host string, port uint32) error {
	args := new(litrpc.ConnectArgs)
	args.LNAddr = address
	reply := new(litrpc.StatusReply)
	if host != "" {
		args.LNAddr += "@" + host
		if port != 2448 && port != 0 {
			args.LNAddr += ":" + strconv.Itoa(int(port))
		}
	}
	err := c.rpcConn.Call("LitRPC.Connect", args, reply)
	if err != nil {
		return err
	}
	if strings.Index(reply.Status, "connected to peer") == -1 {
		return fmt.Errorf("Unexpected response from server: %s", reply.Status)
	}
	return nil
}
