FROM photon:3.0

ARG SGCTRL_VER=development

RUN tdnf install -y tar wget gzip

# adapted from golang docker image
ENV PATH /usr/local/go/bin:$PATH
ENV GOLANG_VERSION 1.17.1
ENV GO_REL_ARCH linux-amd64
ENV GO_REL_SHA dab7d9c34361dc21ec237d584590d72500652e7c909bf082758fb63064fca0ef

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
# helpful ldflags reference: https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-X 'main.Version=$SGCTRL_VER' -buildid=" -trimpath -o controller ./cmd/controller/...

# --- run ---
FROM photon:3.0

COPY --from=0 /go/src/github.com/vmware-tanzu/carvel-secretgen-controller/controller secretgen-controller

# Run as secretgen-controller by default, will be overridden to a random uid on OpenShift
USER 1000
ENV PATH="/:${PATH}"
ENTRYPOINT ["/secretgen-controller"]
