FROM golang:1.13
WORKDIR /go/src/github.com/vmware-tanzu/carvel-secretgen-controller/

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags=-buildid= -trimpath -o controller ./cmd/controller/...

# ---
FROM ubuntu:bionic

RUN groupadd -g 2000 secretgen-controller && useradd -r -u 1000 --create-home -g secretgen-controller secretgen-controller
USER secretgen-controller

COPY --from=0 /go/src/github.com/vmware-tanzu/carvel-secretgen-controller/controller .

ENV PATH="/:${PATH}"
ENTRYPOINT ["/controller"]
