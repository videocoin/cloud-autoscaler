FROM golang:1.14 as builder

RUN apt-get update
RUN apt-get install -y ca-certificates

COPY . /go/src/github.com/videocoin/cloud-autoscaler

WORKDIR /go/src/github.com/videocoin/cloud-autoscaler

RUN make build

FROM bitnami/minideb:jessie

RUN apt-get update
RUN apt-get install -y ca-certificates

COPY --from=builder /go/src/github.com/videocoin/cloud-autoscaler/bin/autoscaler /autoscaler
COPY --from=builder /go/src/github.com/videocoin/cloud-autoscaler/rules.yml /etc/autoscaler/rules.yml
COPY --from=builder /go/src/github.com/videocoin/cloud-autoscaler/rules.yml /rules.yml

WORKDIR /

CMD ["/autoscaler"]
