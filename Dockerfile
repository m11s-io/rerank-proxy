FROM golang:1.27.0-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/rerank-proxy ./cmd/rerank-proxy

FROM scratch
COPY --from=build /out/rerank-proxy /rerank-proxy
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/rerank-proxy"]
