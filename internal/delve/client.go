package delve

import (
	"context"
	"fmt"
	"net"
	"net/rpc/jsonrpc"

	"github.com/go-delve/delve/service/rpc2"
)

type KubeDelveClient struct {
	*rpc2.RPCClient
	conn net.Conn
}

func NewKubeDelveClient(ctx context.Context, host string, port int) (*KubeDelveClient, error) {
	addr := fmt.Sprintf("%v:%v", host, port)
	_, err := jsonrpc.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	// beware: the use of rpc2.NewClient will land you with a log.Fatal upon error
	return &KubeDelveClient{rpc2.NewClientFromConn(conn), conn}, nil
}

func (kdc KubeDelveClient) ValidateState() error {
	state, err := kdc.GetStateNonBlocking()
	if err != nil {
		return err
	}
	if state.Exited {
		// Exited indicates whether the debugged process has exited.
		return fmt.Errorf("delve debugged process has exited")
	} else {
		// were good
		return nil
	}
}

func (kdc KubeDelveClient) Close() error {
	return kdc.conn.Close()
}
