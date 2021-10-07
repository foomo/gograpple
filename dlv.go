package gograpple

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/sirupsen/logrus"
)

func checkDelveServer(l *logrus.Entry, host string, port int, timeout time.Duration, client *rpc2.RPCClient) (*rpc2.RPCClient, *api.DebuggerState, error) {
	if client == nil {
		// connection timeouts suck with k8s, because the port is open and you can connect, but ...
		// early on the connection is a dead and the timeout does not kick in, despite the fact
		// that the connection is not "really" establised
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%v:%v", host, port), time.Hour)
		if err != nil {
			return nil, nil, err
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
			// this is the actual timeout, because the connetion timeout does not work,
			conn.Close()
			l.Debug("dlv server: connection closed due to timeout", timeout)
			return nil, nil, errors.New("stale connection to dlv, aborting after timeout")
		case <-chanClient:
			// we are good to go
			l.Debug("dlv server: good to go")
		}
	}
	st, errState := client.GetState()
	if errState != nil {
		l.Debug("dlv server: could not get state while using client", errState)
		return nil, nil, errState
	}
	return client, st, nil
}
