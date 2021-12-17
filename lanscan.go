package main

// Ping the whole 192.168.1.*
// For the unpinged, try some other well-known ports,
//     and if they answer, add them to the list of pinged's
// For the pinged's,
//     Get as much info as possible about what is there
//     Check the ports of interest

// 192.168.1.0/24 = range of ip's from 192.168.1.1 to 192.168.1.255
// Class C: 255.255.255.0

import (
	"fmt"
	"net"
	"os"
	S "strings"
)

// MyNetIfcAdrs is (via net.InterfaceAddrs()) a list of the system's unicast
// interface addresses. The returned list does not identify the associated
// interface; use Interfaces() and Interface.Addrs() for more detail.
// net.InterfaceAddrs() ([]net.Addr, error)
var MyNetIfcAdrs []net.Addr

// MyNetIfcs is (via net.Interfaces())
// a list of the system's network interfaces.
// net.Interfaces() ([]net.Interface, error)
var MyNetIfcs []net.Interface

var MyHostname string

// ParseCIDR parses s as IP address and prefix length, like
// "192.0.2.0/24" or "2001:db8::/32", per RFCs 4632 & 4291.
// It returns (a) the IP address and (b) the network, that
// are implied by (a) the IP and (b) the prefix length. For
// example, ParseCIDR("192.0.2.1/24") returns IP address
// 192.0.2.1 and network 192.0.2.0/24.
// net.ParseCIDR(s string) (IP, *IPNet, error)

var MyCidrIP net.IP
var MyCidrIPNet *net.IPNet

func init() {
	var adrs []net.Addr
	var e error
	MyHostname, e = os.Hostname()
	if e != nil {
		panic("init(): os.Hostname(): " + e.Error())
	}
	MyNetIfcs, e = net.Interfaces()
	if e != nil {
		panic("init(): net.Interfaces(): " + e.Error())
	}
	for i, ifc := range MyNetIfcs {
		var name, longName string
		name = ifc.Name
		adrs = AdrsOf(&ifc)
		switch name {
		case "lo0", "lo": // local loopback
			longName = ":loopback"
		case "en0", "eth0": // ethernet primary
			longName = ":ethernet0"
		}
		// fmt.Printf("NetIfc[%d:%s%s][%d] %v \n",
		//	i, name, longName, len(adrs), ifc)
		if longName != "" {
			fmt.Printf("NetIfc[%d:%s%s][%d] \n\t %v \n",
				i, name, longName, len(adrs), ifc)
			for j, adr := range adrs {
				var s, sCand string
				s = adr.String()
				// Is it of the form n.n.n.n/n
				if (name == "en0" || name == "eth0") &&
					S.Count(s, ".") == 3 && S.Count(s, "/") == 1 {
					if sipEN0 != "" {
						println(">> MULTIPLE CANDIDATES")
					}
					sipEN0 = s
					sCand = "(Suitable)"
				}
				fmt.Printf("   adr[%d] %s %s \n", j, s, sCand)
			}
		}
	}
	/*
		MyNetIfcAdrs, e = net.InterfaceAddrs()
		if e != nil {
			panic("init(): net.InterfaceAddrs(): " + e.Error())
		}
		for i, adr := range MyNetIfcAdrs {
			fmt.Printf("NetIfcAdr[%d]: %v \n", i, adr)
		}
	*/
}

var ipOtb net.IP
var ipLkp net.IP
var ipEN0 net.IP
var sipOtb string
var sipLkp string
var sipEN0 string

func main() {
	CheckAndReturnClassC()
}

func CheckAndReturnClassC() string {
	// DoAllNetIfcs()
	// fmt.Println("ResolveHostIp(): ", ResolveHostIp())
	ipOtb = GetOutboundIP()
	ipLkp = LookupHost()
	sipOtb = ipOtb.String()
	sipLkp = ipLkp.String()
	lkpFail := ""
	if S.HasPrefix(sipLkp, "127.") {
		lkpFail = "(lookup failed)"
	}
	fmt.Printf("Outbound (wrt.UDP): %s \n", sipOtb)
	fmt.Printf("LookupIP(Hostname): %s %s \n", sipLkp, lkpFail)
	fmt.Printf("ethernet-0 Class C: %s \n", sipEN0)
	if lkpFail != "" {
		sipLkp = sipOtb
	} else if sipOtb != sipLkp {
		panic("Outbound and Lookup do not match")
	}
	if !S.HasPrefix(sipEN0, sipOtb) {
		panic("en0 network does not match others")
	}
	if !S.HasSuffix(sipEN0, "/24") {
		panic("en0network is wrong size (i.e. is not \"/24\"")
	}
	if !S.HasPrefix(sipEN0, "192.168.") {
		panic("en0 does not appear to be normal Class C (192.168.x.x)")
	}
	println("Sanity checks succeeded.")
	return sipEN0
}

func AdrsOf(pIfc *net.Interface) []net.Addr {
	// Addrs returns a list of unicast interface addresses for a specific interface.
	// func (ifi *net.Interface) Addrs() ([]Addr, error)
	adrs, e := pIfc.Addrs()
	if e != nil {
		panic(fmt.Sprintf("AdrsOf(netIfc:%v): %s", *pIfc, e))
	}
	return adrs
}

// Get preferred outbound ip of this machine.
// Is UDP, so no connection is actually established.
// Any outgoing address can be used.
func GetOutboundIP() net.IP {
	// The second parameter can be any IP address except 127.0.0.1
	conn, err := net.Dial("udp", "8.8.8.8:80")
	// or: conn,err := net.Dial("ip:icmp","google.com")
	if err != nil {
		panic("getOtbIP(): " + err.Error())
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

// Loop thru all network interfaces
func DoAllNetIfcs() {
	var adrs []net.Addr
	var adr net.Addr
	for _, ifc := range MyNetIfcs {
		adrs = AdrsOf(&ifc)
		for _, adr = range adrs {
			var ip net.IP
			switch v := adr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			fmt.Printf(">> %T %v \n", ip, ip)
		}
		// process IP address
	}
}

// net.LookupHost() on your os.Hostname() is probably
// always going to give you 127.0.0.1, because that's
// what's in your /etc/hosts or equivalent.
// I think what you want to use is net.InterfaceAddrs()

// This worked for me. Unlike the poster's example,
// it returns only non-loopback addresses, e.g. 10.120.X.X
func LookupHost() net.IP {
	host, _ := os.Hostname()
	addrs, _ := net.LookupIP(host)
	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			return ipv4 // fmt.Sprintf("%T ", ipv4) + ipv4.String()
		}
	}
	return nil
}

func ResolveHostIp() string {
	for _, netInterfaceAddress := range MyNetIfcAdrs {
		networkIp, ok := netInterfaceAddress.(*net.IPNet)
		if ok && !networkIp.IP.IsLoopback() && networkIp.IP.To4() != nil {
			strIP := networkIp.IP.String()
			fmt.Println("IPNet: Resolved Host IP: " + strIP)
			return strIP
		}
	}
	return ""
}
