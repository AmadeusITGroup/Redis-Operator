package utils

import (
	"net"
)

// GetMyIPs return IPs
func GetMyIPs() ([]string, error) {
	var outputAddrs []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return outputAddrs, err
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				outputAddrs = append(outputAddrs, ipnet.IP.String())
			}
		}
	}
	return outputAddrs, nil
}
