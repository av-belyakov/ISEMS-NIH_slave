package common

import (
	"encoding/json"

	"ISEMS-NIH_slave/configure"
)

//NotifyParameters параметры необходимые для пеедачи сообщения
type NotifyParameters struct {
	ClientID, TaskID string
	ChanRes          chan<- configure.MsgWsTransmission
}

//SendMsgErr отправляет информацию об ошибках
func (np *NotifyParameters) SendMsgErr(errName, errDesc string) error {
	msg := configure.MsgTypeError{
		MsgType: "error",
		Info: configure.DetailInfoMsgError{
			TaskID:           np.TaskID,
			ErrorName:        errName,
			ErrorDescription: errDesc,
		},
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	np.ChanRes <- configure.MsgWsTransmission{
		ClientID: np.ClientID,
		Data:     &msgJSON,
	}

	return nil
}

//SendMsgNotify отправляет информационные сообщения
func (np *NotifyParameters) SendMsgNotify(notifyType, section, notifyDesc, typeActionPerform string) error {
	msg := configure.MsgTypeNotification{
		MsgType: "notification",
		Info: configure.DetailInfoMsgNotification{
			TaskID:              np.TaskID,
			Section:             section,
			TypeActionPerformed: typeActionPerform,
			CriticalityMessage:  notifyType,
			Description:         notifyDesc,
		},
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	np.ChanRes <- configure.MsgWsTransmission{
		ClientID: np.ClientID,
		Data:     &msgJSON,
	}

	return nil
}
