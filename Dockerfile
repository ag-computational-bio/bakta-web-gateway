FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM golang:latest as builder

RUN mkdir /BaktaWebGateway
WORKDIR /BaktaWebGateway
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o BaktaWebGateway .

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /BaktaWebGateway/BaktaWebGateway .
COPY config/config.yaml /
WORKDIR /www
COPY www .

ENTRYPOINT [ "/BaktaWebGateway", "-c", "/config.yaml" ]