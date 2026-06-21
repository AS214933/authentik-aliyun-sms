# authentik-aliyun-sms

Go middleware for connecting authentik's Generic SMS provider to Alibaba Cloud SMS.

authentik posts a JSON payload to this service. The service maps that payload to Alibaba Cloud Dysmsapi and returns a non-2xx status when Alibaba Cloud rejects the SMS, so authentik can treat the send as failed.

## Endpoints

- `GET /healthz`
- `POST /send`

authentik Generic provider body:

```json
{
  "To": "+8613800138000",
  "Body": "123456"
}
```

`From` is optional. `Message` is also accepted as a fallback field for the text body.

Phone numbers may be sent in E.164 form such as `+8613800138000`; the service removes the leading `+` and common separators before calling Alibaba Cloud because Dysmsapi examples use dialing-code-prefixed digits. Mainland mobile numbers provided as `13800138000` are sent as `8613800138000`.

If `AUTH_TOKEN` is set, configure authentik to send:

```text
Authorization: Bearer <AUTH_TOKEN>
```

## Configuration

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `HTTP_ADDR` | no | `:8080` | HTTP listen address. |
| `AUTH_TOKEN` | no | empty | Optional bearer token required on `/send`. |
| `ALIYUN_ACCESS_KEY_ID` | yes | empty | Alibaba Cloud AccessKey ID. |
| `ALIYUN_ACCESS_KEY_SECRET` | yes | empty | Alibaba Cloud AccessKey secret. |
| `ALIYUN_ENDPOINT` | no | `dysmsapi-xman.cn-hongkong.aliyuncs.com` | Dysmsapi endpoint. |
| `ALIYUN_REGION_ID` | no | `cn-hongkong` | Region ID sent to Alibaba Cloud. This must match a Dysmsapi-supported region, otherwise Alibaba Cloud returns `InvalidRegion`. |
| `ALIYUN_TIMEOUT_SECONDS` | no | `10` | Per-message send timeout. |
| `ALIYUN_SMS_MODE` | no | `auto` | `auto`, `mainland`, or `global`. |
| `ALIYUN_SMS_SIGN_NAME` | mainland | empty | SMS signature name for template sends; sent to Alibaba Cloud as the `From` parameter. |
| `ALIYUN_SMS_TEMPLATE_CODE` | mainland/auto | empty | Alibaba Cloud SMS template code. |
| `ALIYUN_SMS_TEMPLATE_PARAM` | mainland/auto | empty | Template variable receiving authentik's code, for example `code`. |
| `ALIYUN_SMS_FROM` | no | empty | Default sender ID. Authentik's `From` overrides this in global mode when provided. |

Modes:

- `global` uses `SendMessageToGlobe` and sends authentik's `Body` as the direct message text.
- `mainland` uses `SendMessageWithTemplate` and wraps authentik's `Body` as `{"<ALIYUN_SMS_TEMPLATE_PARAM>":"<Body>"}`. Alibaba Cloud's 2018-05-01 SDK names the mainland signature parameter `From`; this service sets it from `ALIYUN_SMS_SIGN_NAME` first.
- `auto` chooses `mainland` for Chinese mainland mobile numbers and `global` for all other numbers.

For mainland template mode, if authentik sends a JSON object in `Body`, that object is passed through as `TemplateParams`. This supports templates with multiple variables.

## Run Locally

```bash
go test ./...
go run ./cmd/authentik-aliyun-sms
```

Example request:

```bash
curl -X POST http://localhost:8080/send \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer change-me' \
  -d '{"From":"authentik","To":"+8613800138000","Body":"123456"}'
```

## Docker

```bash
docker build -t authentik-aliyun-sms .
docker run --rm -p 8080:8080 \
  -e AUTH_TOKEN=change-me \
  -e ALIYUN_ACCESS_KEY_ID=your-access-key-id \
  -e ALIYUN_ACCESS_KEY_SECRET=your-access-key-secret \
  -e ALIYUN_SMS_MODE=auto \
  -e ALIYUN_SMS_SIGN_NAME=your-sign-name \
  -e ALIYUN_SMS_TEMPLATE_CODE=SMS_123456789 \
  -e ALIYUN_SMS_TEMPLATE_PARAM=code \
  authentik-aliyun-sms
```

`docker-compose.yml` contains the same settings as a starter deployment.

## authentik Setup

In the authentik SMS stage, select the Generic provider and configure:

- URL: `https://<this-service>/send`
- Method: `POST`
- Body: `{"To":"{{ phone_number }}","Body":"{{ code }}"}`
- Headers: `Authorization: Bearer <AUTH_TOKEN>` if `AUTH_TOKEN` is configured

authentik treats HTTP status codes `400` and above as provider failures. This service returns `502` when Alibaba Cloud rejects or fails the SMS request.

## GitHub Container Registry

The GitHub Actions workflow in `.github/workflows/docker.yml`:

- runs `go test ./...`
- builds `linux/amd64` and `linux/arm64` images
- pushes to `ghcr.io/<owner>/<repo>` on branch and tag pushes
- skips pushing for pull requests

The workflow uses `GITHUB_TOKEN` with `packages: write`; no extra GHCR secret is required for the same repository.
