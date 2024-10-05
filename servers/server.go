package servers

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/woodylan/go-websocket/pkg/setting"
	"github.com/woodylan/go-websocket/tools/util"
	"net/http"
	"time"
)

//channel通道
var ToClientChan chan clientInfo

//channel通道结构体
type clientInfo struct {
	ClientId   string
	SendUserId string
	MessageId  string
	Code       int
	Msg        string
	Data       *string
}

type RetData struct {
	MessageId  string      `json:"messageId"`
	SendUserId string      `json:"sendUserId"`
	Code       int         `json:"code"`
	Msg        string      `json:"msg"`
	Data       interface{} `json:"data"`
}

// 心跳间隔,服务端主动循环ping所有客户端,此处放在到2分钟
// 一般客户端主动ping服务器端可以在15-45秒之间
var clientHeartbeatInterval uint64 = 35
var serverHeartbeatInterval = 120 * time.Second

func init() {
	ToClientChan = make(chan clientInfo, 1000)
}

var Manager = NewClientManager() // 管理者

func StartWebSocket() {
	websocketHandler := &Controller{}
	http.HandleFunc("/ws", websocketHandler.Run)

	go Manager.Start()
}

//发送信息到指定客户端
func SendMessage2Client(clientId string, sendUserId string, code int, msg string, data *string) (messageId string) {
	messageId = util.GenUUID()
	if util.IsCluster() {
		addr, _, _, isLocal, err := util.GetAddrInfoAndIsLocal(clientId)
		if err != nil {
			log.Errorf("%s", err)
			return
		}

		//如果是本机则发送到本机
		if isLocal {
			SendMessage2LocalClient(messageId, clientId, sendUserId, code, msg, data)
		} else {
			//发送到指定机器
			SendRpc2Client(addr, messageId, sendUserId, clientId, code, msg, data)
		}
	} else {
		//如果是单机服务，则只发送到本机
		SendMessage2LocalClient(messageId, clientId, sendUserId, code, msg, data)
	}

	return
}

//关闭客户端
func CloseClient(clientId, systemId string) {
	if util.IsCluster() {
		addr, _, _, isLocal, err := util.GetAddrInfoAndIsLocal(clientId)
		if err != nil {
			log.Errorf("%s", err)
			return
		}

		//如果是本机则发送到本机
		if isLocal {
			CloseLocalClient(clientId, systemId)
		} else {
			//发送到指定机器
			CloseRpcClient(addr, clientId, systemId)
		}
	} else {
		//如果是单机服务，则只发送到本机
		CloseLocalClient(clientId, systemId)
	}

	return
}

//设置客户端扩展字段
func SetClientExtend(systemId string, clientId string, userId string, extend string) {
	//如果是集群则用redis共享数据
	if util.IsCluster() {
		//判断key是否存在
		addr, _, _, isLocal, err := util.GetAddrInfoAndIsLocal(clientId)
		if err != nil {
			log.Errorf("%s", err)
			return
		}

		if isLocal {
			if client, err := Manager.GetByClientId(clientId); err == nil {
				//本地
				Manager.SetClientExtend(client, userId, extend)
			} else {
				log.Error(err)
			}
		} else {
			//发送到指定的机器
			SendRpcSetExtend(addr, systemId, clientId, userId, extend)
		}
	} else {
		if client, err := Manager.GetByClientId(clientId); err == nil {
			//如果是单机，就直接本地修改
			Manager.SetClientExtend(client, userId, extend)
		}
	}
}

//添加客户端到分组
func AddClient2Group(systemId string, groupName string, clientId string, userId string, extend string) {
	//如果是集群则用redis共享数据
	if util.IsCluster() {
		//判断key是否存在
		addr, _, _, isLocal, err := util.GetAddrInfoAndIsLocal(clientId)
		if err != nil {
			log.Errorf("%s", err)
			return
		}

		if isLocal {
			if client, err := Manager.GetByClientId(clientId); err == nil {
				//添加到本地
				Manager.AddClient2LocalGroup(groupName, client, userId, extend)
			} else {
				log.Error(err)
			}
		} else {
			//发送到指定的机器
			SendRpcBindGroup(addr, systemId, groupName, clientId, userId, extend)
		}
	} else {
		if client, err := Manager.GetByClientId(clientId); err == nil {
			//如果是单机，就直接添加到本地group了
			Manager.AddClient2LocalGroup(groupName, client, userId, extend)
		}
	}
}

//发送信息到指定分组
func SendMessage2Group(systemId, sendUserId, groupName string, code int, msg string, data *string) (messageId string) {
	messageId = util.GenUUID()
	if util.IsCluster() {
		//发送分组消息给指定广播
		go SendGroupBroadcast(systemId, messageId, sendUserId, groupName, code, msg, data)
	} else {
		//如果是单机服务，则只发送到本机
		Manager.SendMessage2LocalGroup(systemId, messageId, sendUserId, groupName, code, msg, data)
	}
	return
}

//发送信息到指定系统
func SendMessage2System(systemId, sendUserId string, code int, msg string, data string) (messageId string) {
	messageId = util.GenUUID()
	if util.IsCluster() {
		//发送到系统广播
		go SendSystemBroadcast(systemId, messageId, sendUserId, code, msg, &data)
	} else {
		//如果是单机服务，则只发送到本机
		Manager.SendMessage2LocalSystem(systemId, messageId, sendUserId, code, msg, &data)
	}
	return
}

//获取分组列表
func GetOnlineList(systemId *string, groupName *string) map[string]interface{} {
	var clientList []string
	if util.IsCluster() {
		//发送到系统广播
		clientList = GetOnlineListBroadcast(systemId, groupName)
	} else {
		//如果是单机服务，则只发送到本机
		retList := Manager.GetGroupClientList(util.GenGroupKey(*systemId, *groupName))
		clientList = append(clientList, retList...)
	}

	return map[string]interface{}{
		"count": len(clientList),
		"list":  clientList,
	}
}

//通过本服务器发送信息
func SendMessage2LocalClient(messageId, clientId string, sendUserId string, code int, msg string, data *string) {
	log.WithFields(log.Fields{
		"host":     setting.GlobalSetting.LocalHost,
		"port":     setting.CommonSetting.HttpPort,
		"clientId": clientId,
	}).Info("发送到通道")
	ToClientChan <- clientInfo{ClientId: clientId, MessageId: messageId, SendUserId: sendUserId, Code: code, Msg: msg, Data: data}
	return
}

//发送关闭信号
func CloseLocalClient(clientId, systemId string) {
	if conn, err := Manager.GetByClientId(clientId); err == nil && conn != nil {
		if conn.SystemId != systemId {
			return
		}
		Manager.DisConnect <- conn
		log.WithFields(log.Fields{
			"host":     setting.GlobalSetting.LocalHost,
			"port":     setting.CommonSetting.HttpPort,
			"clientId": clientId,
		}).Info("主动踢掉客户端")
	}
	return
}

//监听并发送给客户端信息
func WriteMessage() {
	for {
		clientInfo := <-ToClientChan

		log.WithFields(log.Fields{
			"host":       setting.GlobalSetting.LocalHost,
			"port":       setting.CommonSetting.HttpPort,
			"clientId":   clientInfo.ClientId,
			"messageId":  clientInfo.MessageId,
			"sendUserId": clientInfo.SendUserId,
			"code":       clientInfo.Code,
			"msg":        clientInfo.Msg,
			"data":       clientInfo.Data,
		}).Info("发送到本机")

		if conn, err := Manager.GetByClientId(clientInfo.ClientId); err == nil && conn != nil {
			if err := Render(conn.Socket, clientInfo.MessageId, clientInfo.SendUserId, clientInfo.Code, clientInfo.Msg, clientInfo.Data); err != nil {
				Manager.DisConnect <- conn
				log.WithFields(log.Fields{
					"host":     setting.GlobalSetting.LocalHost,
					"port":     setting.CommonSetting.HttpPort,
					"clientId": clientInfo.ClientId,
					"msg":      clientInfo.Msg,
				}).Error("客户端异常离线：" + err.Error())
			}
		}
	}
}

func Render(conn *websocket.Conn, messageId string, sendUserId string, code int, message string, data interface{}) error {
	return conn.WriteJSON(RetData{
		Code:       code,
		MessageId:  messageId,
		SendUserId: sendUserId,
		Msg:        message,
		Data:       data,
	})
}

/*
//启动定时器进行心跳检测
func PingTimer() {
	go func() {
		ticker := time.NewTicker(serverHeartbeatInterval)
		defer ticker.Stop()
		for {
			<-ticker.C
			//发送心跳
			for clientId, conn := range Manager.AllClient() {
				log.Debugf("心跳检查: %s", clientId)
				if err := conn.Socket.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
					Manager.DisConnect <- conn
					log.Errorf("发送心跳失败: %s 总连接数：%d", clientId, Manager.Count())
				} else {
					if client, err := Manager.GetByClientId(clientId); err == nil {
						Manager.ClientIdMap[clientId].LastTime = uint64(time.Now().Unix())

						log.WithFields(log.Fields{
							"clientId": client.ClientId,
							"userId":	client.UserId,
							"systemId": client.SystemId,
							"ConnectTime":  client.ConnectTime,
						}).Warn("心跳检查")
					}
				}
			}
		}

	}()
}
*/


//启动定时器进行心跳检测
func PingTimer() {
	go func() {
		ticker := time.NewTicker(serverHeartbeatInterval)
		defer ticker.Stop()
		for {
			<-ticker.C
			//发送心跳
			for clientId, conn := range Manager.AllClient() {
				log.Debugf("心跳检查: %s", clientId)

				if uint64(time.Now().Unix()) - Manager.ClientIdMap[clientId].LastTime > clientHeartbeatInterval {
					if err := conn.Socket.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err == nil {
						//当客户端的ping过期时,此处服务器端主动ping一次
						//通时更新时间,不通时断开
						log.Debugf("服务器主动Ping: %s", clientId)
						Manager.resetClientLastTime(conn)
					} else {
						Manager.DisConnect <- conn
						log.Warnf("发送心跳失败: %s 总连接数：%d", clientId, Manager.Count())
					}
				}
			}
		}

	}()
}
