package system

import (
	"encoding/binary"
	"fmt"
	"net"
)

func CalculateNetworkAndBroadcast(ipAddress, netmask string) (string, string, int, error) {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return "", "", 0, fmt.Errorf("Invalid IP '%s'", ipAddress)
	}

	mask := net.ParseIP(netmask)
	if mask == nil {
		return "", "", 0, fmt.Errorf("Invalid netmask '%s'", netmask)
	}

	ip = ip.To4()
	mask = mask.To4()

	if ip != nil && mask != nil {
		return calculateV4NetworkAndBroadcast(ip, mask)
	}

	return "", "", 0, nil
}

func calculateV4NetworkAndBroadcast(ipAddress, netmask net.IP) (string, string, int, error) {
	mask := net.IPMask(netmask)
	broadcast := make(net.IP, net.IPv4len)

	binary.BigEndian.PutUint32(broadcast,
		binary.BigEndian.Uint32(ipAddress.To4())|^binary.BigEndian.Uint32(netmask.To4()))

	network := ipAddress.Mask(mask)
	if network == nil {
		return "", "", 0, fmt.Errorf("could not apply mask %v to IP address %v", mask, ipAddress)
	}

	maskSize, _ := mask.Size()

	return network.To4().String(), broadcast.To4().String(), maskSize, nil
}
