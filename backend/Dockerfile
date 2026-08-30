# 构建阶段
FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/server ./cmd/server

# 运行阶段
FROM alpine:3.20
RUN adduser -D app
USER app
COPY --from=build /bin/server /bin/server
EXPOSE 8080
ENTRYPOINT ["/bin/server"]
