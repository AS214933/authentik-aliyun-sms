# syntax=docker/dockerfile:1

FROM golang:1.22-bookworm AS build
WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/authentik-aliyun-sms ./cmd/authentik-aliyun-sms

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/authentik-aliyun-sms /authentik-aliyun-sms
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/authentik-aliyun-sms"]
