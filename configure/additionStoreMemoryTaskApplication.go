package configure

import (
	"fmt"
	"time"
)

//managemetRecordClientSetting управление учетом настроек клиетов
func managemetRecordClientSettings(sma *StoreMemoryApplication, msg chanReqSettingsTask) chanResSettingsTask {
	funcName := "(func 'managemetRecordClientSettings')"

	switch msg.ActionType {
	case "set":
		settings, ok := msg.Parameters.(ClientSettings)
		if !ok {
			return chanResSettingsTask{
				Error: fmt.Errorf("type conversion error %v", funcName),
			}
		}

		sma.clientSettings[msg.ClientID] = settings

		if _, ok := sma.clientTasks[msg.ClientID]; !ok {
			sma.clientTasks[msg.ClientID] = TasksList{
				filtrationTasks: map[string]*FiltrationTasks{},
				downloadTasks:   map[string]*DownloadTasks{},
			}
		}

		return chanResSettingsTask{}

	case "get":
		if err := sma.checkExistClientSetting(msg.ClientID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		csi, _ := sma.clientSettings[msg.ClientID]

		return chanResSettingsTask{Parameters: csi}

	case "get all":
		for id, cs := range sma.clientSettings {
			if cs.IP == msg.ClientIP {
				return chanResSettingsTask{
					Error: nil,
					Parameters: ClientSettings{
						ID:                id,
						IP:                cs.ID,
						Port:              cs.Port,
						ConnectionStatus:  cs.ConnectionStatus,
						Token:             cs.Token,
						DateLastConnected: cs.DateLastConnected,
						AccessIsAllowed:   cs.AccessIsAllowed,
						SendsTelemetry:    cs.SendsTelemetry,
					},
				}
			}
		}

		return chanResSettingsTask{
			Error: fmt.Errorf("information for client c ip address '%v' not found (func 'managemetRecordClientSettings')", msg.ClientIP),
		}

	case "del":
		delete(sma.clientSettings, msg.ClientID)

		return chanResSettingsTask{}

	case "change connection":
		if err := sma.checkExistClientSetting(msg.ClientID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		status, ok := msg.Parameters.(bool)

		if !ok {
			return chanResSettingsTask{Error: fmt.Errorf("format conversion error")}
		}

		cs := sma.clientSettings[msg.ClientID]
		cs.ConnectionStatus = status

		if status {
			cs.DateLastConnected = time.Now().Unix()
		} else {
			cs.AccessIsAllowed = false
		}

		sma.clientSettings[msg.ClientID] = cs

		return chanResSettingsTask{}
	}

	return chanResSettingsTask{}
}

//managemetRecordTaskFiltration управление учетом задач по фильтрации
func managemetRecordTaskFiltration(sma *StoreMemoryApplication, msg chanReqSettingsTask) chanResSettingsTask {
	switch msg.ActionType {
	case "get task information":
		if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		return chanResSettingsTask{
			Parameters: sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID],
		}

	case "inc num proc files":
		if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		num := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberProcessedFiles + 1
		sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberProcessedFiles = num

		return chanResSettingsTask{Parameters: num}

	case "inc num not proc files":
		if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		num := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberErrorProcessedFiles + 1
		sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberErrorProcessedFiles = num

		return chanResSettingsTask{Parameters: num}

	case "inc num found files":
		if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		num := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberFilesFoundResultFiltering + 1
		sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberFilesFoundResultFiltering = num

		if fileSize, ok := msg.Parameters.(int64); ok {
			size := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].SizeFilesFoundResultFiltering + fileSize
			sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].SizeFilesFoundResultFiltering = size
		}

		return chanResSettingsTask{Parameters: num}

	case "set information task filtration":
		if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		settings, ok := msg.Parameters.(map[string]interface{})

		if !ok {
			return chanResSettingsTask{Error: fmt.Errorf("type conversion error")}
		}

		for k, v := range settings {
			switch k {
			case "FileStorageDirectory":
				if fsd, ok := v.(string); ok {
					sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].FileStorageDirectory = fsd
				}

			case "NumberFilesMeetFilterParameters", "NumberProcessedFiles", "NumberErrorProcessedFiles":
				if count, ok := v.(int); ok {
					if k == "NumberFilesMeetFilterParameters" {
						sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberFilesMeetFilterParameters = count
					}

					if k == "NumberProcessedFiles" {
						sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberProcessedFiles = count
					}

					if k == "NumberErrorProcessedFiles" {
						sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberErrorProcessedFiles = count
					}
				}

			case "SizeFilesMeetFilterParameters":
				if size, ok := v.(int64); ok {
					sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].SizeFilesMeetFilterParameters = size
				}

			case "Status":
				if status, ok := v.(string); ok {
					sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].Status = status
				}

			default:
				return chanResSettingsTask{Error: fmt.Errorf("you cannot change the value, undefined passed parameter")}
			}
		}

	case "add new chan to stop filtration":
		if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		if csf, ok := msg.Parameters.(chan struct{}); ok {
			lcsf := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].ListChanStopFiltration

			lcsf = append(lcsf, csf)
			sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].ListChanStopFiltration = lcsf
		}

	case "delete task":
		resMsg := chanResSettingsTask{}
		if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
			resMsg.Error = err
		}

		delete(sma.clientTasks[msg.ClientID].filtrationTasks, msg.TaskID)

		return resMsg
	}

	return chanResSettingsTask{}
}

//managemetRecordTaskDownload управление учетом задач по скачиванию файлов
func managemetRecordTaskDownload(sma *StoreMemoryApplication, msg chanReqSettingsTask) chanResSettingsTask {
	switch msg.ActionType {
	case "add chan stop read file":
		if err := sma.checkForDownloadingTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		if csrf, ok := msg.Parameters.(chan struct{}); ok {
			tid := sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID]
			tid.ChanStopReadFile = csrf

			sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID] = tid
		}

	case "close chan stop read file":
		if err := sma.checkForDownloadingTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		tid := sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID]
		chanStop := tid.ChanStopReadFile

		tid.ChanStopReadFile = nil
		sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID] = tid

		close(chanStop)

	case "get task information":
		var crst chanResSettingsTask

		if err := sma.checkForDownloadingTask(msg.ClientID, msg.TaskID); err != nil {
			crst.Error = err
		} else {
			crst.Parameters = sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID]
		}

		return crst

	case "get all task information":
		return chanResSettingsTask{
			Parameters: sma.clientTasks[msg.ClientID].downloadTasks,
		}

	case "set is completed task download":
		if err := sma.checkForDownloadingTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		tid := sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID]
		tid.IsTaskCompleted = true
		sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID] = tid

	case "inc num chunk sent":
		if err := sma.checkForDownloadingTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		num := sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID].NumChunkSent + 1
		sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID].NumChunkSent = num

		return chanResSettingsTask{
			Parameters: num,
		}

	case "delete task":
		if err := sma.checkForDownloadingTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		delete(sma.clientTasks[msg.ClientID].downloadTasks, msg.TaskID)

	case "delete all tasks":
		if err := sma.checkForDownloadingTask(msg.ClientID, msg.TaskID); err != nil {
			return chanResSettingsTask{Error: err}
		}

		for taskID := range sma.clientTasks[msg.ClientID].downloadTasks {
			delete(sma.clientTasks[msg.ClientID].downloadTasks, taskID)
		}
	}

	return chanResSettingsTask{}
}
