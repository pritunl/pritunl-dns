package networks

import (
	"github.com/dropbox/godropbox/errors"
	"net"
	"time"
)

var (
	Networks   = map[string]*net.IPNet{}
	lastUpdate = time.Time{}
)

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

		if i == 0 && time.Since(lastUpdate) > 30*time.Second {
			lastUpdate = time.Now()
			Update()
		}
	}

	return ""
}
