package setextend

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
	ClientId  string `json:"clientId" validate:"required"`
	UserId    string `json:"userId"`
	Extend    string `json:"extend"` // 拓展字段，方便业务存储数据
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
	servers.SetClientExtend(systemId, inputData.ClientId, inputData.UserId, inputData.Extend)

	api.Render(w, retcode.SUCCESS, "success", []string{})
}
