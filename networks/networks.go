package networks

import (
	"net"
	"strings"
	"time"

	"github.com/dropbox/godropbox/errors"
)

var (
	Networks = map[string]*net.IPNet{}
)

func init() {
	go func() {
		for {
			Update()
			time.Sleep(30 * time.Second)
		}
	}()
}

func Update() (err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		err = &SystemError{
			errors.Wrap(err, "networks: Error getting interfaces"),
		}
		return
	}

	networks := map[string]*net.IPNet{}

	for _, iface := range ifaces {
		if !strings.Contains(iface.Name, "tun") &&
			!strings.Contains(iface.Name, "wg") {

			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			_, subnet, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}

			networks[subnet.String()] = subnet
		}
	}

	Networks = networks

	return
}

func Find(ip net.IP) string {
	for i := 0; i < 2; i++ {
		for subnetStr, subnet := range Networks {
			if subnet.Contains(ip) {
				return subnetStr
			}
		}
	}

	return ""
}
