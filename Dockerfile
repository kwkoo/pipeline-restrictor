FROM golang:1.14.4 as builder

ARG PREFIX=github.com/kwkoo
ARG PACKAGE=pipelinerestrictor
LABEL builder=true
COPY src /go/src/
RUN \
  set -x \
  && \
  cd /go/src/${PREFIX}/${PACKAGE}/cmd/${PACKAGE} \
  && \
  CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/${PACKAGE} .

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

LABEL maintainer="kin.wai.koo@gmail.com"
LABEL builder=false
COPY --from=builder /go/bin/${PACKAGE} /usr/bin/${PACKAGE}

RUN chmod 755 /usr/bin/${PACKAGE}

USER 1001

ENTRYPOINT ["/usr/bin/pipelinerestrictor"]
CMD ["--tls-cert-file", "/certificates/tls.crt", "--tls-private-key-file", "/certificates/tls.key", "--secure-port", "8443", "--logtostderr"]
