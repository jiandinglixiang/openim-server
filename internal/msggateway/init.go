package msggateway

import (
	"fmt"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/config"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/constant"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/log"
	"time"
)

func RunWsAndServer(rpcPort, wsPort, prometheusPort int) error {
	log.NewPrivateLog(constant.LogFileName)
	fmt.Println("start rpc/msg_gateway server, port: ", rpcPort, wsPort, prometheusPort, ", OpenIM version: ", config.Version)
	longServer, err := NewWsServer(
		WithPort(wsPort),
		WithMaxConnNum(int64(config.Config.LongConnSvr.WebsocketMaxConnNum)),
		WithHandshakeTimeout(time.Duration(config.Config.LongConnSvr.WebsocketTimeOut)*time.Second),
		WithMessageMaxMsgLength(config.Config.LongConnSvr.WebsocketMaxMsgLen))
	if err != nil {
		return err
	}
	hubServer := NewServer(rpcPort, longServer)
	go hubServer.Start()
	hubServer.LongConnServer.Run()
	return nil
}