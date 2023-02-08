package actions

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/foomo/gograpple"
)

type HostPort struct {
	Host string
	Port int
}

func NewHostPort(host string, port int) *HostPort {
	addr, err := gograpple.CheckTCPConnection(host, port)
	if err == nil {
		host = addr.IP.String()
		port = addr.Port
	}
	return &HostPort{host, port}
}

func (lf *HostPort) Set(value string) error {
	pieces := strings.Split(value, ":")
	switch true {
	case len(pieces) == 1 && pieces[0] != value:
		lf.Host = "127.0.0.1"
		var err error
		lf.Port, err = strconv.Atoi(pieces[0])
		if err != nil {
			return err
		}
	case len(pieces) == 2:
		lf.Host = pieces[0]
		if pieces[0] == "" {
			lf.Host = "127.0.0.1"
		}
		var err error
		lf.Port, err = strconv.Atoi(pieces[1])
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid address %q provided", value)
	}
	addr, err := gograpple.CheckTCPConnection(lf.Host, lf.Port)
	if err != nil {
		return err
	}
	lf.Host = addr.IP.String()
	lf.Port = addr.Port
	return nil
}

func (lf HostPort) String() string {
	return fmt.Sprintf("%v:%v", lf.Host, lf.Port)
}

func (*HostPort) Type() string {
	return "host:port"
}

type StringList struct {
	separator string
	items     []string
}

func NewStringList(separator string) *StringList {
	return &StringList{separator: separator}
}

func (sl *StringList) Set(value string) error {
	if value == "" {
		return nil
	}
	sl.items = strings.Split(value, sl.separator)
	return nil
}

func (sl StringList) String() string {
	return strings.Join(sl.items, sl.separator)
}

func (sl *StringList) Type() string {
	return fmt.Sprintf("string separated: %q", sl.separator)
}
