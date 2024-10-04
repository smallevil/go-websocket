package main

import (
	"fmt"
	"strings"
	"github.com/woodylan/go-websocket/define"
	"github.com/woodylan/go-websocket/pkg/etcd"
	"github.com/woodylan/go-websocket/pkg/setting"
	"github.com/woodylan/go-websocket/routers"
	"github.com/woodylan/go-websocket/servers"
	"github.com/woodylan/go-websocket/tools/log"
	"github.com/woodylan/go-websocket/tools/util"
	"net"
	"net/http"
)

func init() {
	setting.Setup()
	log.Setup(false)
}

func main() {
	//初始化RPC服务
	initRPCServer()

	//将服务器地址、端口注册到etcd中
	registerServer()

	//初始化路由
	routers.Init()

	//启动一个定时器用来发送心跳
	servers.PingTimer()

	fmt.Printf("服务器启动成功，端口号：%s\n", setting.CommonSetting.HttpPort)

	//注册默认项目
	restoreSystemId()

	if err := http.ListenAndServe(":"+setting.CommonSetting.HttpPort, nil); err != nil {
		panic(err)
	}
}

func restoreSystemId() {

	for _, value := range setting.ProjectSetting.Systemid{
		systemId := ""
		hookUrl := ""

		s1 := strings.Split(value, "[")
		if len(s1) == 2 {
			s2 := strings.Split(s1[1], "]")
			systemId = s1[0]
			hookUrl = s2[0]
		} else {
			systemId = s1[0]
		}

		fmt.Printf("默认注册系统项目  system_id:%s  hool_url:%s\n", systemId, hookUrl)

		servers.Register(systemId, hookUrl)
	}
}

func initRPCServer() {
	//如果是集群，则启用RPC进行通讯
	if util.IsCluster() {
		//初始化RPC服务
		servers.InitGRpcServer()
		fmt.Printf("启动RPC，端口号：%s\n", setting.CommonSetting.RPCPort)
	}
}

//ETCD注册发现服务
func registerServer() {
	if util.IsCluster() {
		//注册租约
		ser, err := etcd.NewServiceReg(setting.EtcdSetting.Endpoints, 5)
		if err != nil {
			panic(err)
		}

		hostPort := net.JoinHostPort(setting.GlobalSetting.LocalHost, setting.CommonSetting.RPCPort)
		//添加key
		err = ser.PutService(define.ETCD_SERVER_LIST+hostPort, hostPort)
		if err != nil {
			panic(err)
		}

		cli, err := etcd.NewClientDis(setting.EtcdSetting.Endpoints)
		if err != nil {
			panic(err)
		}
		_, err = cli.GetService(define.ETCD_SERVER_LIST)
		if err != nil {
			panic(err)
		}
	}
}
