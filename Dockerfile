FROM golang:1.13
WORKDIR /go/src/github.com/k14s/secretgen-controller/

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o controller ./cmd/controller/...

# ---
FROM ubuntu:bionic

RUN groupadd -g 2000 secretgen-controller && useradd -r -u 1000 --create-home -g secretgen-controller secretgen-controller
USER secretgen-controller

COPY --from=0 /go/src/github.com/k14s/secretgen-controller/controller .

ENV PATH="/:${PATH}"
ENTRYPOINT ["/controller"]
