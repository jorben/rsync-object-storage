FROM --platform=$BUILDPLATFORM golang:1.20-alpine as build
WORKDIR /app
ADD . .
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o ros -ldflags="-w -s"

FROM --platform=$TARGETPLATFORM alpine

WORKDIR /app
COPY --from=build /app/ros ./ros

CMD ["/app/ros", "-c", "/app/config.yaml"]