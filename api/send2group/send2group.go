package send2group

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
	SendUserId string `json:"sendUserId"`
	GroupName  string `json:"groupName"`
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	Data       string `json:"data"`
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

	messageId := ""
	systemId := r.Header.Get("SystemId")
	if len(inputData.GroupName) > 0 {
		messageId = servers.SendMessage2Group(systemId, inputData.SendUserId, inputData.GroupName, inputData.Code, inputData.Msg, &inputData.Data)
	} else {
		messageId = servers.SendMessage2System(systemId, inputData.SendUserId, inputData.Code, inputData.Msg, inputData.Data)
	}


	api.Render(w, retcode.SUCCESS, "success", map[string]string{
		"messageId": messageId,
	})
	return
}
