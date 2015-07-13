package networks

import (
	"net"
	"github.com/dropbox/godropbox/errors"
)

var Networks = map[string]*net.IPNet{}

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
	for subnetStr, subnet := range Networks {
		if subnet.Contains(ip) {
			return subnetStr
		}
	}

	return ""
}
