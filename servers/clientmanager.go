package servers

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/woodylan/go-websocket/define/retcode"
	"github.com/woodylan/go-websocket/pkg/setting"
	"github.com/woodylan/go-websocket/tools/util"
	"sync"
	"time"
	"strconv"
	"net/http"
	"net/url"
	"strings"
	"io/ioutil"
	//"fmt"
)

// 连接管理
type ClientManager struct {
	ClientIdMap     map[string]*Client // 全部的连接
	ClientIdMapLock sync.RWMutex       // 读写锁

	Connect    chan *Client // 连接处理
	DisConnect chan *Client // 断开连接处理

	GroupLock sync.RWMutex
	Groups    map[string][]string

	SystemClientsLock sync.RWMutex
	SystemClients     map[string][]string
}

type pongHandler func(string) error

func NewClientManager() (clientManager *ClientManager) {
	clientManager = &ClientManager{
		ClientIdMap:   make(map[string]*Client),
		Connect:       make(chan *Client, 10000),
		DisConnect:    make(chan *Client, 10000),
		Groups:        make(map[string][]string, 100),
		SystemClients: make(map[string][]string, 100),
	}

	return
}

// 管道处理程序
func (manager *ClientManager) Start() {
	for {
		select {
		case client := <-manager.Connect:
			// 建立连接事件
			manager.EventConnect(client)
		case conn := <-manager.DisConnect:
			// 断开连接事件
			manager.EventDisconnect(conn)
		}
	}
}

// 建立连接事件
func (manager *ClientManager) EventConnect(client *Client) {
	manager.AddClient(client)

	client.Socket.SetPingHandler(func(string) error {
		//log.Print("PING")
		//manager.ClientIdMapLock.Lock()
		//defer manager.ClientIdMapLock.Unlock()
		//manager.ClientIdMap[client.ClientId].LastTime = uint64(time.Now().Unix())
		manager.resetClientLastTime(client)

		return client.Socket.WriteMessage(websocket.PongMessage, []byte(""))
	})

	CallHookUrl(client, "online")

	log.WithFields(log.Fields{
		"host":     setting.GlobalSetting.LocalHost,
		"port":     setting.CommonSetting.HttpPort,
		"clientId": client.ClientId,
		"userId":	client.UserId,
		"systemId": client.SystemId,
		"counts":   Manager.Count(),
	}).Info("客户端已连接")
}

// 断开连接时间
func (manager *ClientManager) EventDisconnect(client *Client) {

	CallHookUrl(client, "offline")

	//关闭连接
	_ = client.Socket.Close()
	manager.DelClient(client)

	mJson, _ := json.Marshal(map[string]string{
		"clientId": client.ClientId,
		"userId":   client.UserId,
		"extend":   client.Extend,
	})
	data := string(mJson)
	sendUserId := ""

	//发送下线通知
	if len(client.GroupList) > 0 {
		for _, groupName := range client.GroupList {
			SendMessage2Group(client.SystemId, sendUserId, groupName, retcode.OFFLINE_MESSAGE_CODE, "客户端下线", &data)
		}
	}

	log.WithFields(log.Fields{
		"host":     setting.GlobalSetting.LocalHost,
		"port":     setting.CommonSetting.HttpPort,
		"clientId": client.ClientId,
		"userId":	client.UserId,
		"systemId": client.SystemId,
		"counts":   Manager.Count(),
		"seconds":  uint64(time.Now().Unix()) - client.ConnectTime,
	}).Info("客户端已断开")

	//标记销毁
	client.IsDeleted = true
	client = nil
}

// 更新client最后活跃时间
func (manager *ClientManager) resetClientLastTime(client *Client) {
	manager.ClientIdMapLock.Lock()
	defer manager.ClientIdMapLock.Unlock()

	client.LastTime = uint64(time.Now().Unix())
}

// 添加客户端
func (manager *ClientManager) AddClient(client *Client) {
	manager.ClientIdMapLock.Lock()
	defer manager.ClientIdMapLock.Unlock()

	manager.ClientIdMap[client.ClientId] = client
}

// 获取所有的客户端
func (manager *ClientManager) AllClient() map[string]*Client {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()

	return manager.ClientIdMap
}

// 客户端数量
func (manager *ClientManager) Count() int {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()

	return len(manager.ClientIdMap)
}

// 删除客户端
func (manager *ClientManager) DelClient(client *Client) {
	manager.delClientIdMap(client.ClientId)

	//删除所在的分组
	if len(client.GroupList) > 0 {
		for _, groupName := range client.GroupList {
			manager.delGroupClient(util.GenGroupKey(client.SystemId, groupName), client.ClientId)
		}
	}

	// 删除系统里的客户端
	manager.delSystemClient(client)
}

// 删除clientIdMap
func (manager *ClientManager) delClientIdMap(clientId string) {
	manager.ClientIdMapLock.Lock()
	defer manager.ClientIdMapLock.Unlock()

	delete(manager.ClientIdMap, clientId)
}

// 通过clientId获取
func (manager *ClientManager) GetByClientId(clientId string) (*Client, error) {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()

	if client, ok := manager.ClientIdMap[clientId]; !ok {
		return nil, errors.New("客户端不存在")
	} else {
		return client, nil
	}
}

// 发送到本机分组
func (manager *ClientManager) SendMessage2LocalGroup(systemId, messageId, sendUserId, groupName string, code int, msg string, data *string) {
	if len(groupName) > 0 {
		clientIds := manager.GetGroupClientList(util.GenGroupKey(systemId, groupName))
		if len(clientIds) > 0 {
			for _, clientId := range clientIds {
				if _, err := Manager.GetByClientId(clientId); err == nil {
					//添加到本地
					SendMessage2LocalClient(messageId, clientId, sendUserId, code, msg, data)
				} else {
					//删除分组
					manager.delGroupClient(util.GenGroupKey(systemId, groupName), clientId)
				}
			}
		}
	}
}

//发送给指定业务系统
func (manager *ClientManager) SendMessage2LocalSystem(systemId, messageId string, sendUserId string, code int, msg string, data *string) {
	if len(systemId) > 0 {
		clientIds := Manager.GetSystemClientList(systemId)
		if len(clientIds) > 0 {
			for _, clientId := range clientIds {
				SendMessage2LocalClient(messageId, clientId, sendUserId, code, msg, data)
			}
		}
	}
}

// 设置扩展字段值
func (manager *ClientManager) SetClientExtend(client *Client, userId string, extend string) {
	manager.ClientIdMapLock.Lock()
	defer manager.ClientIdMapLock.Unlock()

	if len(userId) > 0 {
		client.UserId = userId
	}

	if len(extend) > 0 {
		client.Extend = extend
	}
}

// 添加到本地分组
func (manager *ClientManager) AddClient2LocalGroup(groupName string, client *Client, userId string, extend string) {
	//标记当前客户端的userId

	if len(userId) > 0 {
		client.UserId = userId
	}

	if len(extend) > 0 {
		client.Extend = extend
	}

	//判断之前是否有添加过
	for _, groupValue := range client.GroupList {
		if groupValue == groupName {
			return
		}
	}

	// 为属性添加分组信息
	groupKey := util.GenGroupKey(client.SystemId, groupName)

	manager.addClient2Group(groupKey, client)

	client.GroupList = append(client.GroupList, groupName)

	mJson, _ := json.Marshal(map[string]string{
		"clientId": client.ClientId,
		"userId":   client.UserId,
		"extend":   client.Extend,
	})
	data := string(mJson)
	sendUserId := ""

	//发送系统通知
	SendMessage2Group(client.SystemId, sendUserId, groupName, retcode.ONLINE_MESSAGE_CODE, "客户端上线", &data)
}

// 添加到本地分组
func (manager *ClientManager) addClient2Group(groupKey string, client *Client) {
	manager.GroupLock.Lock()
	defer manager.GroupLock.Unlock()
	manager.Groups[groupKey] = append(manager.Groups[groupKey], client.ClientId)
}

// 删除分组里的客户端
func (manager *ClientManager) delGroupClient(groupKey string, clientId string) {
	manager.GroupLock.Lock()
	defer manager.GroupLock.Unlock()

	for index, groupClientId := range manager.Groups[groupKey] {
		if groupClientId == clientId {
			manager.Groups[groupKey] = append(manager.Groups[groupKey][:index], manager.Groups[groupKey][index+1:]...)
		}
	}
}

// 获取本地分组的成员
func (manager *ClientManager) GetGroupClientList(groupKey string) []string {
	manager.GroupLock.RLock()
	defer manager.GroupLock.RUnlock()
	return manager.Groups[groupKey]
}

// 添加到系统客户端列表
func (manager *ClientManager) AddClient2SystemClient(systemId string, client *Client) {
	manager.SystemClientsLock.Lock()
	defer manager.SystemClientsLock.Unlock()
	manager.SystemClients[systemId] = append(manager.SystemClients[systemId], client.ClientId)
}

// 删除系统里的客户端
func (manager *ClientManager) delSystemClient(client *Client) {
	manager.SystemClientsLock.Lock()
	defer manager.SystemClientsLock.Unlock()

	for index, clientId := range manager.SystemClients[client.SystemId] {
		if clientId == client.ClientId {
			manager.SystemClients[client.SystemId] = append(manager.SystemClients[client.SystemId][:index], manager.SystemClients[client.SystemId][index+1:]...)
		}
	}
}

// 获取指定系统的客户端列表
func (manager *ClientManager) GetSystemClientList(systemId string) []string {
	manager.SystemClientsLock.RLock()
	defer manager.SystemClientsLock.RUnlock()
	return manager.SystemClients[systemId]
}

// hook回调
func CallHookUrl(client *Client, status string) {
	/*
	ai, ok := SystemMap.Load(client.SystemId)
	fmt.Println(ai)
	fmt.Println(ok)
	fmt.Println(client)
	*/

	var ai accountInfo
	v, ok := SystemMap.Load(client.SystemId)
	if !ok {
		return
	}
	ai = v.(accountInfo)

	if len(ai.HookUrl) == 0 {
		return
	}

	clientID := client.ClientId
	userID := client.UserId
	extend := client.Extend
	ip := client.Ip
	connectTime := client.ConnectTime
	lastTime := client.LastTime

	values := url.Values{}
	values.Add("system_id", ai.SystemId)
	values.Add("client_id", clientID)
	values.Add("user_id", userID)
	values.Add("extend", extend)
	values.Add("ip", ip)
	values.Add("connect_time", strconv.FormatUint(connectTime, 10))
	values.Add("last_time", strconv.FormatUint(lastTime, 10))
	values.Add("keep_seconds", strconv.FormatUint(lastTime - connectTime, 10))
	values.Add("status", status)

	httpClient := http.Client{
		Timeout:   2 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	resp, err := httpClient.Post(ai.HookUrl, "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
	if err != nil {
		log.WithFields(log.Fields{
			"host":     ai.HookUrl,
			"clientId": clientID,
			"ip": 		ip,
			"userId":	userID,
			"extend":	extend,
			"systemId": ai.SystemId,
			"status":	status,
			"keep_seconds":  uint64(time.Now().Unix()) - connectTime,
			"error":	err,
		}).Warn("Hook请求失败")
		return
	}

	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"host":     ai.HookUrl,
			"clientId": clientID,
			"ip": 		ip,
			"userId":	userID,
			"extend":	extend,
			"systemId": ai.SystemId,
			"status":	status,
			"keep_seconds":  uint64(time.Now().Unix()) - connectTime,
			"error":	err,
		}).Warn("Hook读取失败")
		return
	}
}
