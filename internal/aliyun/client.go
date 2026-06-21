package aliyun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20180501 "github.com/alibabacloud-go/dysmsapi-20180501/v2/client"
	"github.com/alibabacloud-go/tea/dara"
)

type Mode string

const (
	ModeAuto     Mode = "auto"
	ModeMainland Mode = "mainland"
	ModeGlobal   Mode = "global"
)

var (
	mainlandNumberPattern      = regexp.MustCompile(`^(?:86)?1[3-9]\d{9}$`)
	localMainlandNumberPattern = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

type Config struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
	Mode            Mode
	SignName        string
	TemplateCode    string
	TemplateParam   string
	From            string
	TimeoutSeconds  int
}

type Message struct {
	From string
	To   string
	Body string
}

type Sender interface {
	Send(ctx context.Context, msg Message) error
}

type Client struct {
	cfg Config
	api dysmsAPI
}

type dysmsAPI interface {
	SendMessageToGlobeWithOptions(request *dysmsapi20180501.SendMessageToGlobeRequest, runtime *dara.RuntimeOptions) (*dysmsapi20180501.SendMessageToGlobeResponse, error)
	SendMessageWithTemplateWithOptions(request *dysmsapi20180501.SendMessageWithTemplateRequest, runtime *dara.RuntimeOptions) (*dysmsapi20180501.SendMessageWithTemplateResponse, error)
}

func (m Mode) Valid() bool {
	return m == ModeAuto || m == ModeMainland || m == ModeGlobal
}

func NewClient(cfg Config) (*Client, error) {
	openAPIConfig := &openapi.Config{
		AccessKeyId:     dara.String(cfg.AccessKeyID),
		AccessKeySecret: dara.String(cfg.AccessKeySecret),
		Endpoint:        dara.String(cfg.Endpoint),
	}

	api, err := dysmsapi20180501.NewClient(openAPIConfig)
	if err != nil {
		return nil, err
	}
	return NewClientWithAPI(cfg, api), nil
}

func NewClientWithAPI(cfg Config, api dysmsAPI) *Client {
	return &Client{
		cfg: cfg,
		api: api,
	}
}

func (c *Client) Send(ctx context.Context, msg Message) error {
	msg.To = normalizePhone(msg.To)
	msg.From = strings.TrimSpace(msg.From)
	msg.Body = strings.TrimSpace(msg.Body)

	if msg.To == "" {
		return errors.New("recipient phone number is required")
	}
	if msg.Body == "" {
		return errors.New("message body is required")
	}

	mode := c.cfg.Mode
	if mode == ModeAuto {
		if isMainlandNumber(msg.To) {
			mode = ModeMainland
		} else {
			mode = ModeGlobal
		}
	}

	sendCtx, cancel := context.WithTimeout(ctx, time.Duration(c.cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		switch mode {
		case ModeMainland:
			done <- c.sendMainland(msg)
		case ModeGlobal:
			done <- c.sendGlobal(msg)
		default:
			done <- fmt.Errorf("unsupported sms mode %q", mode)
		}
	}()

	select {
	case <-sendCtx.Done():
		return fmt.Errorf("aliyun sms request timed out: %w", sendCtx.Err())
	case err := <-done:
		return err
	}
}

func (c *Client) sendGlobal(msg Message) error {
	request := &dysmsapi20180501.SendMessageToGlobeRequest{
		To:      dara.String(msg.To),
		Message: dara.String(msg.Body),
	}
	if from := firstNonEmpty(msg.From, c.cfg.From); from != "" {
		request.From = dara.String(from)
	}

	response, err := c.api.SendMessageToGlobeWithOptions(request, &dara.RuntimeOptions{})
	if err != nil {
		return err
	}
	if response == nil || response.Body == nil {
		return errors.New("aliyun returned empty global sms response")
	}
	if dara.StringValue(response.Body.ResponseCode) != "OK" {
		return fmt.Errorf("aliyun global sms failed: code=%s message=%s", dara.StringValue(response.Body.ResponseCode), dara.StringValue(response.Body.ResponseDescription))
	}
	return nil
}

func (c *Client) sendMainland(msg Message) error {
	templateParams, err := buildTemplateParams(c.cfg.TemplateParam, msg.Body)
	if err != nil {
		return err
	}

	request := &dysmsapi20180501.SendMessageWithTemplateRequest{
		To:            dara.String(formatMainlandNumber(msg.To)),
		TemplateCode:  dara.String(c.cfg.TemplateCode),
		TemplateParam: dara.String(templateParams),
	}
	if from := firstNonEmpty(c.cfg.SignName, msg.From, c.cfg.From); from != "" {
		request.From = dara.String(from)
	}

	response, err := c.api.SendMessageWithTemplateWithOptions(request, &dara.RuntimeOptions{})
	if err != nil {
		return err
	}
	if response == nil || response.Body == nil {
		return errors.New("aliyun returned empty mainland sms response")
	}
	if dara.StringValue(response.Body.ResponseCode) != "OK" {
		return fmt.Errorf("aliyun mainland sms failed: code=%s message=%s", dara.StringValue(response.Body.ResponseCode), dara.StringValue(response.Body.ResponseDescription))
	}
	return nil
}

func buildTemplateParams(paramName, body string) (string, error) {
	paramName = strings.TrimSpace(paramName)
	if paramName == "" {
		return "", errors.New("template parameter name is required")
	}

	raw := strings.TrimSpace(body)
	var existing map[string]any
	if json.Valid([]byte(raw)) && json.Unmarshal([]byte(raw), &existing) == nil {
		encoded, err := json.Marshal(existing)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}

	encoded, err := json.Marshal(map[string]string{paramName: raw})
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func isMainlandNumber(number string) bool {
	return mainlandNumberPattern.MatchString(normalizePhone(number))
}

func formatMainlandNumber(number string) string {
	normalized := normalizePhone(number)
	if localMainlandNumberPattern.MatchString(normalized) {
		return "86" + normalized
	}
	return normalized
}

func normalizePhone(number string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")
	return strings.TrimPrefix(replacer.Replace(strings.TrimSpace(number)), "+")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
