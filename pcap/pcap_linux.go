package pcap

import (
	"net"
	"os"

	"github.com/getlantern/golog"
	"github.com/getlantern/ringbuffer"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
)

const (
	snapLen = 1600
)

var (
	log = golog.LoggerFor("pcap")
)

func noop(ip string) {}

// Capture() begins capturing network packets for the named interface and
// returns a function that can be called at any time to dump packets to/from the
// given IP address to a file named "pcaps". If anything goes wrong, this
// function will log errors but doesn't return them to the caller.
func Capture(interfaceName string) (dump func(ip string)) {
	ifAddrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Errorf("Unable to determine interface addresses")
		return noop
	}
	localInterfaces := make(map[string]bool, len(ifAddrs))
	for _, ifAddr := range ifAddrs {
		localInterfaces[ifAddr.String()] = true
	}

	buffersByIP := make(map[string]ringbuffer.RingBuffer)
	getBufferByIP := func(ip string) ringbuffer.RingBuffer {
		buffer := buffersByIP[ip]
		if buffer == nil {
			buffer = ringbuffer.New(100) // todo make this configurable
			buffersByIP[ip] = buffer
		}
		return buffer
	}

	handle, err := pcap.OpenLive(interfaceName, snapLen, true, pcap.BlockForever)
	if err != nil {
		log.Errorf("Unable to open %v for packet capture: %v", interfaceName, err)
		return noop
	}
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	stop := make(chan bool)
	stopped := make(chan bool)

	capturePacket := func(dstIP string, srcIP string, packet gopacket.Packet) {
		if !localInterfaces[dstIP] {
			getBufferByIP(dstIP).Push(packet)
		} else if !localInterfaces[srcIP] {
			getBufferByIP(srcIP).Push(packet)
		}
	}

	doCapture := func() {
		for packet := range packetSource.Packets() {
			nl := packet.NetworkLayer()
			switch t := nl.(type) {
			case *layers.IPv4:
				capturePacket(t.DstIP.String(), t.SrcIP.String(), packet)
			case *layers.IPv6:
				capturePacket(t.DstIP.String(), t.SrcIP.String(), packet)
			}
			select {
			case <-stop:
				stopped <- true
				return
			default:
				// continue
			}
		}
	}

	go doCapture()

	return func(ip string) {
		log.Debugf("Attempting to dump pcaps for %v", ip)

		pcapsFileName := "/tmp/" + ip + ".pcap"
		newFile := false
		pcapsFile, err := os.OpenFile(pcapsFileName, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Errorf("Unable to open pcap file: %v", err)
				return
			}
			pcapsFile, err = os.Create(pcapsFileName)
			if err != nil {
				log.Errorf("Unable to create pcap file: %v", err)
				return
			}
			newFile = true
		}
		pcaps := pcapgo.NewWriter(pcapsFile)
		if newFile {
			pcaps.WriteFileHeader(snapLen, layers.LinkTypeEthernet)
		}

		dumpPacket := func(dstIP string, srcIP string, packet gopacket.Packet) {
			if dstIP == ip || srcIP == ip {
				pcaps.WritePacket(packet.Metadata().CaptureInfo, packet.Data())
			}
		}

		stop <- true
		<-stopped

		getBufferByIP(ip).IterateForward(func(_packet interface{}) bool {
			if _packet == nil {
				// TODO: figure out why we need this guard condition, since we shouldn't
				return false
			}
			packet := _packet.(gopacket.Packet)
			nl := packet.NetworkLayer()
			switch t := nl.(type) {
			case *layers.IPv4:
				dumpPacket(t.DstIP.String(), t.SrcIP.String(), packet)
			case *layers.IPv6:
				dumpPacket(t.DstIP.String(), t.SrcIP.String(), packet)
			}
			return true
		})

		delete(buffersByIP, ip)
		go doCapture()

		pcapsFile.Close()
		log.Debugf("Logged pcaps for %v to %v", ip, pcapsFile.Name())
	}
}
