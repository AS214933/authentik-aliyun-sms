package aliyun

import (
	"context"
	"testing"

	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	dysmsapi20180501 "github.com/alibabacloud-go/dysmsapi-20180501/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/dara"
)

type fakeGlobalAPI struct {
	globalRequest   *dysmsapi20180501.SendMessageToGlobeRequest
	mainlandRequest *dysmsapi20180501.SendMessageWithTemplateRequest
	globalResponse  *dysmsapi20180501.SendMessageToGlobeResponse
	mainlandResp    *dysmsapi20180501.SendMessageWithTemplateResponse
	err             error
}

type fakeDomesticAPI struct {
	request  *dysmsapi20170525.SendSmsRequest
	response *dysmsapi20170525.SendSmsResponse
	err      error
}

func (f *fakeGlobalAPI) SendMessageToGlobeWithOptions(request *dysmsapi20180501.SendMessageToGlobeRequest, _ *dara.RuntimeOptions) (*dysmsapi20180501.SendMessageToGlobeResponse, error) {
	f.globalRequest = request
	return f.globalResponse, f.err
}

func (f *fakeGlobalAPI) SendMessageWithTemplateWithOptions(request *dysmsapi20180501.SendMessageWithTemplateRequest, _ *dara.RuntimeOptions) (*dysmsapi20180501.SendMessageWithTemplateResponse, error) {
	f.mainlandRequest = request
	return f.mainlandResp, f.err
}

func (f *fakeDomesticAPI) SendSmsWithOptions(request *dysmsapi20170525.SendSmsRequest, _ *util.RuntimeOptions) (*dysmsapi20170525.SendSmsResponse, error) {
	f.request = request
	return f.response, f.err
}

func TestSendGlobalMapsMessageToAliyunRequest(t *testing.T) {
	api := &fakeGlobalAPI{
		globalResponse: &dysmsapi20180501.SendMessageToGlobeResponse{
			Body: &dysmsapi20180501.SendMessageToGlobeResponseBody{
				ResponseCode: dara.String("OK"),
			},
		},
	}
	client := NewClientWithAPI(Config{Mode: ModeGlobal, TimeoutSeconds: 1}, api, successfulDomesticAPI())

	err := client.Send(context.Background(), Message{From: "Auth", To: "+15551234567", Body: "123456"})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if dara.StringValue(api.globalRequest.From) != "Auth" {
		t.Fatalf("unexpected From: %q", dara.StringValue(api.globalRequest.From))
	}
	if dara.StringValue(api.globalRequest.To) != "15551234567" {
		t.Fatalf("unexpected To: %q", dara.StringValue(api.globalRequest.To))
	}
	if dara.StringValue(api.globalRequest.Message) != "123456" {
		t.Fatalf("unexpected Message: %q", dara.StringValue(api.globalRequest.Message))
	}
}

func TestSendDomesticBuildsTemplateParams(t *testing.T) {
	api := &fakeDomesticAPI{
		response: &dysmsapi20170525.SendSmsResponse{
			Body: &dysmsapi20170525.SendSmsResponseBody{
				Code: dara.String("OK"),
			},
		},
	}
	client := NewClientWithAPI(Config{
		Mode:           ModeMainland,
		TemplateCode:   "SMS_123",
		TemplateParam:  "code",
		SignName:       "Example",
		TimeoutSeconds: 1,
	}, &fakeGlobalAPI{}, api)

	err := client.Send(context.Background(), Message{To: "13800138000", Body: "654321"})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if api.request == nil {
		t.Fatal("expected domestic request")
	}
	if dara.StringValue(api.request.TemplateCode) != "SMS_123" {
		t.Fatalf("unexpected template code: %q", dara.StringValue(api.request.TemplateCode))
	}
	if dara.StringValue(api.request.PhoneNumbers) != "13800138000" {
		t.Fatalf("unexpected PhoneNumbers: %q", dara.StringValue(api.request.PhoneNumbers))
	}
	if dara.StringValue(api.request.TemplateParam) != `{"code":"654321"}` {
		t.Fatalf("unexpected template params: %q", dara.StringValue(api.request.TemplateParam))
	}
	if dara.StringValue(api.request.SignName) != "Example" {
		t.Fatalf("unexpected SignName: %q", dara.StringValue(api.request.SignName))
	}
}

func TestAutoModeChoosesMainlandForChinaMobileNumber(t *testing.T) {
	globalAPI := &fakeGlobalAPI{}
	domesticAPI := successfulDomesticAPI()
	client := NewClientWithAPI(Config{
		Mode:           ModeAuto,
		TemplateCode:   "SMS_123",
		TemplateParam:  "code",
		SignName:       "Example",
		TimeoutSeconds: 1,
	}, globalAPI, domesticAPI)

	err := client.Send(context.Background(), Message{To: "+8613800138000", Body: "654321"})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if domesticAPI.request == nil {
		t.Fatal("expected domestic request")
	}
	if globalAPI.globalRequest != nil {
		t.Fatal("did not expect global request")
	}
}

func TestMainlandSignNameOverridesAuthentikFrom(t *testing.T) {
	api := successfulDomesticAPI()
	client := NewClientWithAPI(Config{
		Mode:           ModeMainland,
		TemplateCode:   "SMS_123",
		TemplateParam:  "code",
		SignName:       "ApprovedSign",
		TimeoutSeconds: 1,
	}, &fakeGlobalAPI{}, api)

	err := client.Send(context.Background(), Message{From: "authentik", To: "13800138000", Body: "654321"})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if dara.StringValue(api.request.SignName) != "ApprovedSign" {
		t.Fatalf("unexpected SignName: %q", dara.StringValue(api.request.SignName))
	}
}

func TestBuildTemplateParamsAcceptsJSONBody(t *testing.T) {
	got, err := buildTemplateParams("code", `{"code":"123456","ttl":"5"}`)
	if err != nil {
		t.Fatalf("build params failed: %v", err)
	}
	if got != `{"code":"123456","ttl":"5"}` {
		t.Fatalf("unexpected params: %s", got)
	}
}

func successfulDomesticAPI() *fakeDomesticAPI {
	return &fakeDomesticAPI{
		response: &dysmsapi20170525.SendSmsResponse{
			Body: &dysmsapi20170525.SendSmsResponseBody{
				Code: dara.String("OK"),
			},
		},
	}
}
