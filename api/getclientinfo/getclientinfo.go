package getclientinfo

import (
	"encoding/json"
	"github.com/woodylan/go-websocket/api"
	"github.com/woodylan/go-websocket/define/retcode"
	"github.com/woodylan/go-websocket/servers"
	"net/http"
)

type Controller struct {
}

type inputData struct {
	ClientId  string      `json:"clientId" validate:"required"`
}

type ClientInfo struct {
	ClientId    string    `json:"client_id"`      // 标识ID
	SystemId    string    `json:"system_id"`      // 系统ID
	Ip    		string    `json:"ip"`      // 客户端IP:PORT
	ConnectTime uint64    `json:"connect_time"`      // 首次连接时间
	LastTime	uint64    `json:"last_time"`      // 最后活跃时间
	UserId      string    `json:"user_id"`      // 业务端标识用户ID
	Extend      string    `json:"extend"`     // 扩展字段，用户可以自定义
	GroupList   []string  `json:"group_list"`
}

func (c *Controller) Run(w http.ResponseWriter, r *http.Request) {
	var inputData inputData
	if err := json.NewDecoder(r.Body).Decode(&inputData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := api.Validate(inputData)
	if err != nil {
		api.Render(w, retcode.FAIL, err.Error(), []string{})
		return
	}

	systemId := r.Header.Get("SystemId")
	ret := servers.GetClientInfo(inputData.ClientId, systemId)


	if ret == nil {
		api.Render(w, retcode.SUCCESS, "success", []string{})
	} else {

		if ret.SystemId != systemId {
			api.Render(w, retcode.SUCCESS, "success", []string{})
		} else {
			var clientInfo ClientInfo

			clientInfo.ClientId = ret.ClientId
			clientInfo.SystemId = ret.SystemId
			clientInfo.Ip = ret.Ip
			clientInfo.ConnectTime = ret.ConnectTime
			clientInfo.LastTime = ret.LastTime
			clientInfo.UserId = ret.UserId
			clientInfo.Extend = ret.Extend
			clientInfo.GroupList = []string{}
			if len(ret.GroupList) > 0 {
				clientInfo.GroupList = ret.GroupList
			}

			api.Render(w, retcode.SUCCESS, "success", clientInfo)
		}
	}

	//api.Render(w, retcode.SUCCESS, "success", ret)
	return
}
