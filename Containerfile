FROM golang:1.22.0-bullseye AS build

WORKDIR /workspace

# Copy the source
COPY ./* /work/
WORKDIR /work/

RUN CGO_ENABLED=1 GOOS=linux GO111MODULE=on go build -a -installsuffix cgo -o kleidi-plugin main.go

FROM registry.access.redhat.com/ubi8/ubi-micro:latest

LABEL org.opencontainers.image.title "Kleidi - Kubernetes KMS Provider Plugin" 
LABEL org.opencontainers.image.vendor "beeyz.dev" 
LABEL org.opencontainers.image.licenses "Apache-2.0 License" 
LABEL org.opencontainers.image.source "https://github.com/beezy-dev/kleidi" 
LABEL org.opencontainers.image.description "Kleidi is an open-source Kubernetes Provider Plugin supporting multiple KMS services." 
LABEL org.opencontainers.image.documentation "https://beezy.dev/kleidi/"

COPY --from=build ./opt/app-root/src/kleidi-plugin .

ENTRYPOINT [ "./kleidi-plugin" ]
