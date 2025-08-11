package controller

import (
	"encoding/json"
	"fmt"
	"myflowhub/pkg/database"
	"myflowhub/pkg/protocol"
	"myflowhub/server/internal/hub"
	"myflowhub/server/internal/service"
	"strings"

	"github.com/rs/zerolog/log"
	"gorm.io/datatypes"
)

// VariableController 负责处理变量相关的消息
type VariableController struct {
	service       *service.VariableService
	deviceService *service.DeviceService
}

// NewVariableController 创建一个新的 VariableController
func NewVariableController(service *service.VariableService, deviceService *service.DeviceService) *VariableController {
	return &VariableController{
		service:       service,
		deviceService: deviceService,
	}
}

// HandleVarsQuery 处理变量查询请求
func (c *VariableController) HandleVarsQuery(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	var payload protocol.VarsQueryPayload
	jsonPayload, _ := json.Marshal(msg.Payload)
	json.Unmarshal(jsonPayload, &payload)

	results := make([]interface{}, len(payload.Queries))
	for i, query := range payload.Queries {
		var deviceIdentifier, varName string
		if strings.Contains(query, ".") {
			parts := strings.SplitN(query, ".", 2)
			deviceIdentifier = parts[0]
			varName = parts[1]
		} else {
			deviceIdentifier = fmt.Sprintf("[%d]", client.DeviceID)
			varName = query
		}

		variable, err := c.service.GetVariableByDeviceUIDAndVarName(deviceIdentifier, varName)
		if err != nil {
			results[i] = nil
			continue
		}

		var val interface{}
		json.Unmarshal(variable.Value, &val)
		results[i] = val
	}

	s.SendResponse(client, msg.ID, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"results": results,
		},
	})
}

// HandleVarUpdate 处理变量更新请求
func (c *VariableController) HandleVarUpdate(s *hub.Server, client *hub.Client, msg protocol.BaseMessage) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}

	variables, ok := payload["variables"].(map[string]interface{})
	if !ok {
		return
	}

	var updatedCount int
	for fqdn, value := range variables {
		var deviceIdentifier, varName string
		if strings.Contains(fqdn, ".") {
			parts := strings.SplitN(fqdn, ".", 2)
			deviceIdentifier = parts[0]
			varName = parts[1]
		} else {
			deviceIdentifier = fmt.Sprintf("[%d]", client.DeviceID)
			varName = fqdn
		}

		if !hub.IsValidVarName(varName) {
			log.Warn().Str("varName", varName).Msg("无效的变量名，已跳过")
			continue
		}

		targetDevice, err := c.deviceService.GetDeviceByUIDOrName(deviceIdentifier)
		if err != nil {
			continue
		}

		jsonValue, _ := json.Marshal(value)
		variable := &database.DeviceVariable{
			OwnerDeviceID: targetDevice.ID,
			VariableName:  varName,
			Value:         datatypes.JSON(jsonValue),
		}
		if c.service.UpsertVariable(variable) == nil {
			updatedCount++
		}
	}
	log.Info().Int("count", updatedCount).Msg("变量已更新")
}
