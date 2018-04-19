package utils

import (
	"net"

	log "github.com/sirupsen/logrus"
)

func ProbeLocalIP(remoteHost string) (IP string, err error) {

	addr, err := net.ResolveUDPAddr("udp4", remoteHost)
	if nil != err {
		log.Warn("resolve zk udp addr:", err)
		return
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if nil != err {
		log.Warn("dial zk udp :", err)
		return
	}

	localAddr, err := net.ResolveUDPAddr("udp4", conn.LocalAddr().String())
	if nil != err {
		log.Warn("resolve local udp addr:", err)
		return
	}

	IP = localAddr.IP.String()

	if err = conn.Close(); nil != err {
		return
	}

	return
}
