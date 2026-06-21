package aliyun

import (
	"context"
	"testing"

	dysmsapi20180501 "github.com/alibabacloud-go/dysmsapi-20180501/v2/client"
	"github.com/alibabacloud-go/tea/dara"
)

type fakeAPI struct {
	globalRequest   *dysmsapi20180501.SendMessageToGlobeRequest
	mainlandRequest *dysmsapi20180501.SendMessageWithTemplateRequest
	globalResponse  *dysmsapi20180501.SendMessageToGlobeResponse
	mainlandResp    *dysmsapi20180501.SendMessageWithTemplateResponse
	err             error
}

func (f *fakeAPI) SendMessageToGlobeWithOptions(request *dysmsapi20180501.SendMessageToGlobeRequest, _ *dara.RuntimeOptions) (*dysmsapi20180501.SendMessageToGlobeResponse, error) {
	f.globalRequest = request
	return f.globalResponse, f.err
}

func (f *fakeAPI) SendMessageWithTemplateWithOptions(request *dysmsapi20180501.SendMessageWithTemplateRequest, _ *dara.RuntimeOptions) (*dysmsapi20180501.SendMessageWithTemplateResponse, error) {
	f.mainlandRequest = request
	return f.mainlandResp, f.err
}

func TestSendGlobalMapsMessageToAliyunRequest(t *testing.T) {
	api := &fakeAPI{
		globalResponse: &dysmsapi20180501.SendMessageToGlobeResponse{
			Body: &dysmsapi20180501.SendMessageToGlobeResponseBody{
				ResponseCode: dara.String("OK"),
			},
		},
	}
	client := NewClientWithAPI(Config{Mode: ModeGlobal, TimeoutSeconds: 1}, api)

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

func TestSendMainlandBuildsTemplateParams(t *testing.T) {
	api := &fakeAPI{
		mainlandResp: &dysmsapi20180501.SendMessageWithTemplateResponse{
			Body: &dysmsapi20180501.SendMessageWithTemplateResponseBody{
				ResponseCode: dara.String("OK"),
			},
		},
	}
	client := NewClientWithAPI(Config{
		Mode:           ModeMainland,
		TemplateCode:   "SMS_123",
		TemplateParam:  "code",
		SignName:       "Example",
		TimeoutSeconds: 1,
	}, api)

	err := client.Send(context.Background(), Message{To: "13800138000", Body: "654321"})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if api.mainlandRequest == nil {
		t.Fatal("expected mainland request")
	}
	if dara.StringValue(api.mainlandRequest.TemplateCode) != "SMS_123" {
		t.Fatalf("unexpected template code: %q", dara.StringValue(api.mainlandRequest.TemplateCode))
	}
	if dara.StringValue(api.mainlandRequest.To) != "8613800138000" {
		t.Fatalf("unexpected To: %q", dara.StringValue(api.mainlandRequest.To))
	}
	if dara.StringValue(api.mainlandRequest.TemplateParam) != `{"code":"654321"}` {
		t.Fatalf("unexpected template params: %q", dara.StringValue(api.mainlandRequest.TemplateParam))
	}
	if dara.StringValue(api.mainlandRequest.From) != "Example" {
		t.Fatalf("unexpected From: %q", dara.StringValue(api.mainlandRequest.From))
	}
}

func TestAutoModeChoosesMainlandForChinaMobileNumber(t *testing.T) {
	api := &fakeAPI{
		mainlandResp: &dysmsapi20180501.SendMessageWithTemplateResponse{
			Body: &dysmsapi20180501.SendMessageWithTemplateResponseBody{
				ResponseCode: dara.String("OK"),
			},
		},
	}
	client := NewClientWithAPI(Config{
		Mode:           ModeAuto,
		TemplateCode:   "SMS_123",
		TemplateParam:  "code",
		TimeoutSeconds: 1,
	}, api)

	err := client.Send(context.Background(), Message{To: "+8613800138000", Body: "654321"})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if api.mainlandRequest == nil {
		t.Fatal("expected mainland request")
	}
	if api.globalRequest != nil {
		t.Fatal("did not expect global request")
	}
}

func TestMainlandSignNameOverridesAuthentikFrom(t *testing.T) {
	api := &fakeAPI{
		mainlandResp: &dysmsapi20180501.SendMessageWithTemplateResponse{
			Body: &dysmsapi20180501.SendMessageWithTemplateResponseBody{
				ResponseCode: dara.String("OK"),
			},
		},
	}
	client := NewClientWithAPI(Config{
		Mode:           ModeMainland,
		TemplateCode:   "SMS_123",
		TemplateParam:  "code",
		SignName:       "ApprovedSign",
		TimeoutSeconds: 1,
	}, api)

	err := client.Send(context.Background(), Message{From: "authentik", To: "13800138000", Body: "654321"})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if dara.StringValue(api.mainlandRequest.From) != "ApprovedSign" {
		t.Fatalf("unexpected From: %q", dara.StringValue(api.mainlandRequest.From))
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
