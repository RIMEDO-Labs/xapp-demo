FROM onosproject/golang-build:v1.0 as build

ENV GO111MODULE=on
ENV XAPPNAME=xapp-demo

COPY Makefile go.mod go.sum /go/src/github.com/rimedo-labs/xapp-demo/
COPY cmd/ /go/src/github.com/rimedo-labs/xapp-demo/cmd/
COPY vendor/ /go/src/github.com/rimedo-labs/xapp-demo/vendor/
COPY pkg/ /go/src/github.com/rimedo-labs/xapp-demo/pkg/
COPY build/build-tools /go/src/github.com/rimedo-labs/xapp-demo/build/build-tools/
# COPY policytypes /go/src/github.com/rimedo-labs/xapp-demo/policytypes/
# COPY schemas /go/src/github.com/rimedo-labs/xapp-demo/schemas/

RUN cd /go/src/github.com/rimedo-labs/xapp-demo && GOFLAGS=-mod=vendor make build

FROM alpine:3.11
RUN apk add libc6-compat

USER nobody

COPY --from=build /go/src/github.com/rimedo-labs/xapp-demo/build/_output/xapp-demo /usr/local/bin/xapp-demo
# COPY --from=build /go/src/github.com/rimedo-labs/xapp-demo/schemas /data/schemas
# COPY --from=build /go/src/github.com/rimedo-labs/xapp-demo/policytypes /data/policytypes

ENTRYPOINT ["xapp-demo"]
