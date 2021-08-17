FROM photon:3.0

ARG SG_VER=development

RUN tdnf install -y tar wget gzip

# adapted from golang docker image
ENV PATH /usr/local/go/bin:$PATH
ENV GOLANG_VERSION 1.16.5
ENV GO_REL_ARCH linux-amd64
ENV GO_REL_SHA b12c23023b68de22f74c0524f10b753e7b08b1504cb7e417eccebdd3fae49061

RUN set eux; \
    wget -O go.tgz "https://golang.org/dl/go${GOLANG_VERSION}.${GO_REL_ARCH}.tar.gz" --progress=dot:giga; \
    echo "${GO_REL_SHA} go.tgz" | sha256sum -c -; \
    tar -C /usr/local -xzf go.tgz; \
    rm go.tgz; \
    go version

ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

WORKDIR /go/src/github.com/vmware-tanzu/carvel-secretgen-controller/

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags=-buildid= -trimpath -o controller ./cmd/controller/...

# --- run ---
FROM photon:3.0

RUN tdnf install -y shadow-tools

RUN groupadd -g 2000 secretgen-controller && useradd -r -u 1000 --create-home -g secretgen-controller secretgen-controller
RUN chmod g+w /etc/pki/tls/certs/ca-bundle.crt && chgrp secretgen-controller /etc/pki/tls/certs/ca-bundle.crt
USER secretgen-controller

COPY --from=0 /go/src/github.com/vmware-tanzu/carvel-secretgen-controller/controller secretgen-controller

ENV PATH="/:${PATH}"
ENTRYPOINT ["/secretgen-controller"]
