package tun

import (
	"github.com/net-byte/vtun/common/config"
	"github.com/net-byte/vtun/common/netutil"
	"github.com/net-byte/water"
	"log"
	"net"
	"runtime"
	"strconv"
)

// CreateTun creates a tun interface
func CreateTun(config config.Config) (iface *water.Interface) {
	// refer: https://pkg.go.dev/github.com/net-byte/water

	// DeviceType is the type for specifying device types.
	// water.TUN 创建 tun 设备
	c := water.Config{DeviceType: water.TUN}
	// 默认 dn 不指定此处为空 只指定tun的cidr
	if config.DeviceName != "" {
		c.PlatformSpecificParams = water.PlatformSpecificParams{Name: config.DeviceName, Network: config.CIDR}
	} else {
		os := runtime.GOOS
		if os == "windows" {
			c.PlatformSpecificParams = water.PlatformSpecificParams{Name: "vtun", Network: config.CIDR}
		} else {
			c.PlatformSpecificParams = water.PlatformSpecificParams{Network: config.CIDR}
		}
	}

	// 变量 c 的内容如下
	// &{DeviceName: LocalAddr::3000 ServerAddr::3001 ServerIP:172.16.0.1 ServerIPv6:fced:9999::1
	// DNSIP:8.8.8.8 CIDR:172.16.0.10/24 CIDRv6:fced:9999::9999/64 Key:freedom Protocol:udp
	//  WebSocketPath:/freedom ServerMode:false GlobalMode:false Obfs:false Compress:false
	//  MTU:1500 Timeout:30 LocalGateway:192.168.3.1 TLSCertificateFilePath:./certs/server.pem
	//  TLSCertificateKeyFilePath:./certs/server.key TLSSni: TLSInsecureSkipVerify:false}

	// New creates a new TUN/TAP interface using config.
	iface, err := water.New(c)
	if err != nil {
		log.Fatalln("failed to create tun interface:", err)
	}
	log.Printf("interface created %v", iface.Name())

	// 配置tun 设备  cidr mtu 等等
	configTun(config, iface)

	return iface
}

// ConfigTun configures the tun interface
func configTun(config config.Config, iface *water.Interface) {
	os := runtime.GOOS
	// 获取 ipv4地址（比如：192.168.0.1)
	ip, _, err := net.ParseCIDR(config.CIDR)
	if err != nil {
		log.Panicf("error cidr %v", config.CIDR)
	}
	ipv6, _, err := net.ParseCIDR(config.CIDRv6)
	if err != nil {
		log.Panicf("error ipv6 cidr %v", config.CIDRv6)
	}

	if os == "linux" {
		// 设置 mtu
		netutil.ExecCmd("/sbin/ip", "link", "set", "dev", iface.Name(), "mtu", strconv.Itoa(config.MTU))
		// 设置 cidr 172.16.0.10/24
		netutil.ExecCmd("/sbin/ip", "addr", "add", config.CIDR, "dev", iface.Name())
		// 设置 ipv6地址
		netutil.ExecCmd("/sbin/ip", "-6", "addr", "add", config.CIDRv6, "dev", iface.Name())
		// 设置 tun 卡 up
		netutil.ExecCmd("/sbin/ip", "link", "set", "dev", iface.Name(), "up")

		// client模式 config.ServerMode为 false； 启动参数-S 后，config.ServerMode为 true
		if !config.ServerMode && config.GlobalMode {
			// 主网卡的名字，一般为 eth0
			physicalIface := netutil.GetInterface()
			// 如果 -s 指定为 127.0.0.1:3301 则 host 为127.0.0.1
			host, _, err := net.SplitHostPort(config.ServerAddr)
			if err != nil {
				log.Panic("error server address")
			}
			// 如果 -s 指定为 127.0.0.1:3301 则 serverIP 为127.0.0.1
			serverIP := netutil.LookupIP(host)
			if physicalIface != "" && serverIP != nil {
				netutil.ExecCmd("/sbin/ip", "route", "add", "0.0.0.0/1", "dev", iface.Name())
				netutil.ExecCmd("/sbin/ip", "-6", "route", "add", "::/1", "dev", iface.Name())
				netutil.ExecCmd("/sbin/ip", "route", "add", "128.0.0.0/1", "dev", iface.Name())
				netutil.ExecCmd("/sbin/ip", "route", "add", config.DNSIP+"/32", "via", config.LocalGateway, "dev", physicalIface)
				if serverIP.To4() != nil {
					netutil.ExecCmd("/sbin/ip", "route", "add", serverIP.To4().String()+"/32", "via", config.LocalGateway, "dev", physicalIface)
				} else {
					netutil.ExecCmd("/sbin/ip", "-6", "route", "add", serverIP.To16().String()+"/64", "via", config.LocalGateway, "dev", physicalIface)
				}
			}
		}

	} else if os == "darwin" {
		gateway := config.ServerIP
		gateway6 := config.ServerIPv6
		netutil.ExecCmd("ifconfig", iface.Name(), "inet", ip.String(), gateway, "up")
		netutil.ExecCmd("ifconfig", iface.Name(), "inet6", ipv6.String(), gateway6, "up")
		if !config.ServerMode && config.GlobalMode {
			// 取主网卡 eth0
			physicalIface := netutil.GetInterface()
			host, _, err := net.SplitHostPort(config.ServerAddr)
			if err != nil {
				log.Panic("error server address")
			}
			serverIP := netutil.LookupIP(host)
			if physicalIface != "" && serverIP != nil {
				if serverIP.To4() != nil {
					netutil.ExecCmd("route", "add", serverIP.To4().String(), config.LocalGateway)
				} else {
					netutil.ExecCmd("route", "add", "-inet6", serverIP.To16().String(), config.LocalGateway)
				}
				netutil.ExecCmd("route", "add", config.DNSIP, config.LocalGateway)
				netutil.ExecCmd("route", "add", "-inet6", "::/1", "-interface", iface.Name())
				netutil.ExecCmd("route", "add", "0.0.0.0/1", "-interface", iface.Name())
				netutil.ExecCmd("route", "add", "128.0.0.0/1", "-interface", iface.Name())
				netutil.ExecCmd("route", "add", "default", gateway)
				netutil.ExecCmd("route", "change", "default", gateway)
			}
		}
	} else if os == "windows" {
		if !config.ServerMode && config.GlobalMode {
			gateway := config.ServerIP
			host, _, err := net.SplitHostPort(config.ServerAddr)
			if err != nil {
				log.Panic("error server address")
			}
			serverIP := netutil.LookupIP(host)
			if serverIP != nil {
				netutil.ExecCmd("cmd", "/C", "route", "delete", "0.0.0.0", "mask", "0.0.0.0")
				netutil.ExecCmd("cmd", "/C", "route", "add", "0.0.0.0", "mask", "0.0.0.0", gateway, "metric", "6")
				netutil.ExecCmd("cmd", "/C", "route", "add", serverIP.To4().String(), config.LocalGateway, "metric", "5")
				netutil.ExecCmd("cmd", "/C", "route", "add", config.DNSIP, config.LocalGateway, "metric", "5")
			}
		}
	} else {
		log.Printf("not support os %v", os)
	}
	log.Printf("interface configured %v", iface.Name())
}

// ResetTun resets the tun interface
func ResetTun(config config.Config) {
	// reset gateway
	if !config.ServerMode && config.GlobalMode {
		os := runtime.GOOS
		if os == "darwin" {
			netutil.ExecCmd("route", "add", "default", config.LocalGateway)
			netutil.ExecCmd("route", "change", "default", config.LocalGateway)
		} else if os == "windows" {
			netutil.ExecCmd("cmd", "/C", "route", "delete", "0.0.0.0", "mask", "0.0.0.0")
			netutil.ExecCmd("cmd", "/C", "route", "add", "0.0.0.0", "mask", "0.0.0.0", config.LocalGateway, "metric", "6")
		}
	}
}
