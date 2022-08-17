package grpc

import (
	"log"
	"net"
	"time"

	"github.com/golang/snappy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/net-byte/water"
	"github.com/ytwxy99/shawnVpn/common/cache"
	"github.com/ytwxy99/shawnVpn/common/cipher"
	"github.com/ytwxy99/shawnVpn/common/config"
	"github.com/ytwxy99/shawnVpn/common/counter"
	"github.com/ytwxy99/shawnVpn/common/netutil"
	"github.com/ytwxy99/shawnVpn/grpc/proto"
)

// The StreamService is the implementation of the StreamServer interface
type StreamService struct {
	proto.UnimplementedGrpcServeServer
	config config.Config
	iface  *water.Interface
}

// Tunnel implements the StreamServer interface
func (s *StreamService) Tunnel(srv proto.GrpcServe_TunnelServer) error {
	toServer(srv, s.config, s.iface)
	return nil
}

// StartServer starts the grpc server
func StartServer(iface *water.Interface, config config.Config) {
	log.Printf("vtun grpc server started on %v", config.LocalAddr)
	ln, err := net.Listen("tcp", config.LocalAddr)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()
	creds, err := credentials.NewServerTLSFromFile(config.TLSCertificateFilePath, config.TLSCertificateKeyFilePath)
	if err != nil {
		log.Panic(err)
	}
	grpcServer := grpc.NewServer(grpc.Creds(creds))
	proto.RegisterGrpcServeServer(grpcServer, &StreamService{config: config, iface: iface})
	go toClient(config, iface)
	err = grpcServer.Serve(ln)
	if err != nil {
		log.Fatalf("grpc server error: %v", err)
	}
}

// toClient sends packets from tun to grpc
func toClient(config config.Config, iface *water.Interface) {
	packet := make([]byte, 4096)
	for {
		n, err := iface.Read(packet)
		if err != nil || n == 0 {
			continue
		}
		b := packet[:n]
		if key := netutil.GetDstKey(b); key != "" {
			if v, ok := cache.GetCache().Get(key); ok {
				if config.Obfs {
					b = cipher.XOR(b)
				}
				if config.Compress {
					b = snappy.Encode(nil, b)
				}
				v.(proto.GrpcServe_TunnelServer).Send(&proto.PacketData{Data: b})
				counter.IncrWrittenBytes(n)
			}
		}
	}
}

// toServer sends packets from grpc to tun
func toServer(srv proto.GrpcServe_TunnelServer, config config.Config, iface *water.Interface) {
	for {
		packet, err := srv.Recv()
		if err != nil {
			break
		}
		b := packet.Data[:]
		if config.Compress {
			b, err = snappy.Decode(nil, b)
			if err != nil {
				break
			}
		}
		if config.Obfs {
			b = cipher.XOR(b)
		}
		if key := netutil.GetSrcKey(b); key != "" {
			cache.GetCache().Set(key, srv, 10*time.Minute)
			iface.Write(b)
			counter.IncrReadBytes(len(b))
		}
	}
}
