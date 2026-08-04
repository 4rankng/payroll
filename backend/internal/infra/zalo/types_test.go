package zalo

import (
	"encoding/json"
	"testing"
)

func TestSendResultJSONContract(t *testing.T) {
	result := SendResult{
		MsgID:      "msg-123",
		ErrorCode:  -118,
		ErrorMsg:   "Số điện thoại chưa liên kết Zalo",
		HTTPStatus: 200,
	}

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal SendResult: %v", err)
	}

	want := `{"msg_id":"msg-123","error_code":-118,"error_msg":"Số điện thoại chưa liên kết Zalo","http_status":200}`
	if string(payload) != want {
		t.Fatalf("SendResult JSON = %s, want %s", payload, want)
	}
}
