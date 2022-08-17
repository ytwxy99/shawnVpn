package app

import (
	"log"

	"github.com/net-byte/water"
	"github.com/ytwxy99/shawnVpn/common/cipher"
	"github.com/ytwxy99/shawnVpn/common/config"
	"github.com/ytwxy99/shawnVpn/common/netutil"
	"github.com/ytwxy99/shawnVpn/grpc"
	"github.com/ytwxy99/shawnVpn/tls"
	"github.com/ytwxy99/shawnVpn/tun"
	"github.com/ytwxy99/shawnVpn/udp"
	"github.com/ytwxy99/shawnVpn/ws"
)

var _banner = `
    
   /--/    | 
  /  /     |        ---     
    \  \   |---    /   \    \    /\    /   /\    /
   /  /    |   |   \   /     \  /  \  /   /  \  / 
  /__/     |   |    ---\\     \/    \/   /    \/

VPN written by shawn wang

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
	log.Printf("ShawnVpn version %s", app.Version)
	if !app.Config.ServerMode {
		app.Config.LocalGateway = netutil.GetLocalGateway()
	}
	cipher.SetKey(app.Config.Key)
	log.Printf("initialized config: %+v", app.Config)
}

// StartApp starts the app
func (app *Vtun) StartApp() {
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
