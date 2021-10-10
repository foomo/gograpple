package delve

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-delve/delve/service/rpc2"
)

type KubeDelveClient struct {
	*rpc2.RPCClient
}

func NewKubeDelveClient(host string, port int, timeout time.Duration) (*KubeDelveClient, error) {
	// connection timeouts suck with k8s, because the port is open and you can connect, but ...
	// early on the connection is a dead and the timeout does not kick in, despite the fact
	// that the connection is not "really" establised
	var client *rpc2.RPCClient
	conn, err := net.Dial("tcp", fmt.Sprintf("%v:%v", host, port))
	if err != nil {
		return nil, err
	}
	chanClient := make(chan struct{})
	go func() {
		// the next level of suck is the rpc client
		// it does not return an error for its constructor,
		// even though it actually does make a call and that fails
		// and the cient will not do any other calls,
		// because it has shutdown internally, without telling us
		// at least that is what I understand here
		client = rpc2.NewClientFromConn(conn)
		chanClient <- struct{}{}
	}()
	select {
	case <-time.After(timeout):
		// this is the actual timeout,
		// since the connection timeout does not work (on k8s)
		conn.Close()
		// l.Warn("dlv server check timeout", timeout)
		return nil, errors.New("stale connection to dlv, aborting after timeout")
	case <-chanClient:
		// were good to go
	}
	return &KubeDelveClient{client}, nil
}

func (kdc KubeDelveClient) ValidateState() error {
	state, err := kdc.GetState()
	if err != nil {
		return err
	}
	if state.Exited {
		// Exited indicates whether the debugged process has exited.
		return fmt.Errorf("delve debugged process has exited")
	} else if !state.Running {
		// Running is true if the process is running and no other information can be collected.
		// theres a case when you are in a breakpoint on a zombie process
		// dlv will not handle that gracefully
		return fmt.Errorf("delve debugged process is not running (is it a zombie, or has it been halted by a breakpoint)")
	} else {
		// were good
		return nil
	}
}
