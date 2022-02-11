// 프로그램 결산 프로그램
//
// Description : 네트워크 관련 스크립트

package main

import (
	"net"
)

func serviceIPFunc() (string, error) {
	ip := "127.0.0.1"
	ifaces, err := net.Interfaces()
	if err != nil {
		return ip, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue //interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return ip, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return ip, nil
}

func serviceMACAddrFunc() (string, error) {
	ip := "127.0.0.1"
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var mac string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue //interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return ip, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			mac = iface.HardwareAddr.String()
			return mac, nil
		}
	}
	return mac, nil
}
