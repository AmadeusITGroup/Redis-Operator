package utils

import (
	"errors"
	"net"
)

// GetMyIP return the local IP
func GetMyIP() (string, error) {
	addrs, err := net.InterfaceAddrs()

	if err == nil {
		for _, address := range addrs {
			// check the address type and if it is not a loopback the display it
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String(), nil
				}
			}
		}
	}
	err = errors.New("Unable to retrieve the local IP")
	return "", err
}
