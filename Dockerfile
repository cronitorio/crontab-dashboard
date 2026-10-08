FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.2-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY internal/ ./internal/
COPY --from=web /app/web/static ./web/static
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w -X github.com/cronitorio/crontab-dashboard/internal/dashboard.Version=$VERSION" -o /out/crontab-dashboard .
COPY CLI_VERSION CLI_REPOSITORY ./
COPY scripts/fetch-cli.sh ./scripts/
RUN apk add --no-cache curl && ./scripts/fetch-cli.sh "linux_$TARGETARCH" /out/cronitor

FROM alpine:3.22
RUN apk add --no-cache bash ca-certificates dcron tzdata && mkdir -p /etc/cronitor /var/spool/cron/crontabs
COPY --from=build /out/ /usr/local/bin/
COPY docker-entrypoint.sh /usr/local/bin/
ENV CRONTAB_DASHBOARD_CONTAINER=1
EXPOSE 9000
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["--port", "9000"]
