package gograpple

import (
	"net"
	"testing"

	"github.com/sirupsen/logrus"
)

const testNamespace = "test"

func testGrapple(t *testing.T, deployment string) *Grapple {
	g, err := NewGrapple(logrus.NewEntry(logrus.StandardLogger()), testNamespace, deployment)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func setUp(t *testing.T, context string) {
	// set context via kubectl
}

func delveSetUp(t *testing.T, g *Grapple) {
	// todo
	// make build
	// make deploy, use testNamespace
	// patch
}

func testAddr(t *testing.T) *net.TCPAddr {
	addr, err := CheckTCPConnection("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

func TestGrapple_Delve(t *testing.T) {
	g := testGrapple(t, "example")
	delveSetUp(t, g)
	addr := testAddr(t)
	type args struct {
		sourcePath string
		host       string
		port       int
		vscode     bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test", args{"test/app", addr.IP.String(), addr.Port, true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := g.Delve("", "", tt.args.sourcePath, nil, tt.args.host, tt.args.port, tt.args.vscode, false); (err != nil) != tt.wantErr {
				t.Errorf("Grapple.Delve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
