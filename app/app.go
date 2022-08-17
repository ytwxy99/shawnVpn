package app

import (
	"github.com/net-byte/vtun/grpc"
	"github.com/net-byte/vtun/tls"
	"github.com/net-byte/vtun/udp"
	"github.com/net-byte/vtun/ws"
	"log"

	"github.com/net-byte/vtun/common/cipher"
	"github.com/net-byte/vtun/common/config"
	"github.com/net-byte/vtun/common/netutil"
	"github.com/net-byte/vtun/tun"
	"github.com/net-byte/water"
)

var _banner = `
    
   /--/    | 
  /  /     |        ---     
    \  \   |---    /   \    \    /\    /   /\    /
   /  /    |   |   \   /     \  /  \  /   /  \  / 
  /__/     |   |    ---\\     \/    \/   /    \/

Simple VPN written by shawn wang

`

// vtun app struct
type Vtun struct {
	Config  *config.Config
	Version string
	Iface   *water.Interface
}

// InitConfig initializes the config
func (app *Vtun) InitConfig() {
	log.Printf(_banner)
	if !app.Config.ServerMode {
		// client mode
		app.Config.LocalGateway = netutil.GetLocalGateway()
	}
	// 用这个 key 作为代理的权限认证
	cipher.SetKey(app.Config.Key)
	log.Printf("initialized config: %+v", app.Config)
}

// StartApp starts the app
func (app *Vtun) StartApp() {
	// 创建 tun 三层虚拟设备
	app.Iface = tun.CreateTun(*app.Config)
	switch app.Config.Protocol {
	case "udp":
		if app.Config.ServerMode {
			udp.StartServer(app.Iface, *app.Config)
		} else {
			udp.StartClient(app.Iface, *app.Config)
		}
	case "ws", "wss":
		if app.Config.ServerMode {
			ws.StartServer(app.Iface, *app.Config)
		} else {
			ws.StartClient(app.Iface, *app.Config)
		}
	case "tls":
		if app.Config.ServerMode {
			tls.StartServer(app.Iface, *app.Config)
		} else {
			tls.StartClient(app.Iface, *app.Config)
		}
	case "grpc":
		if app.Config.ServerMode {
			grpc.StartServer(app.Iface, *app.Config)
		} else {
			grpc.StartClient(app.Iface, *app.Config)
		}
	default:
		if app.Config.ServerMode {
			udp.StartServer(app.Iface, *app.Config)
		} else {
			udp.StartClient(app.Iface, *app.Config)
		}
	}
}

// StopApp stops the app
func (app *Vtun) StopApp() {
	tun.ResetTun(*app.Config)
	app.Iface.Close()
	log.Println("vtun stopped")
}
