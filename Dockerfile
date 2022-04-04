FROM photon:3.0

ARG SGCTRL_VER=development
ARG TARGETARCH
ARG TARGETOS

RUN tdnf install -y tar wget gzip

# adapted from golang docker image
ENV PATH /usr/local/go/bin:$PATH
ENV GOLANG_VERSION 1.17.1

RUN set eux; \
    wget -O go.tgz "https://golang.org/dl/go${GOLANG_VERSION}.${TARGETOS}-${TARGETARCH}.tar.gz" --progress=dot:giga; \
    if [ $TARGETARCH == "arm64" ] ; then export DOWNLOAD_SHA="53b29236fa03ed862670a5e5e2ab2439a2dc288fe61544aa392062104ac0128c" ; fi; \
    if [ $TARGETARCH == "amd64" ] ; then export DOWNLOAD_SHA="dab7d9c34361dc21ec237d584590d72500652e7c909bf082758fb63064fca0ef" ; fi; \
    echo "${DOWNLOAD_SHA} go.tgz" | sha256sum -c -; \
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

RUN tdnf install -y shadow-tools

RUN groupadd -g 2000 secretgen-controller && useradd -r -u 1000 --create-home -g secretgen-controller secretgen-controller
RUN chmod g+w /etc/pki/tls/certs/ca-bundle.crt && chgrp secretgen-controller /etc/pki/tls/certs/ca-bundle.crt
USER secretgen-controller

COPY --from=0 /go/src/github.com/vmware-tanzu/carvel-secretgen-controller/controller secretgen-controller

ENV PATH="/:${PATH}"
ENTRYPOINT ["/secretgen-controller"]
