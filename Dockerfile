FROM --platform=$BUILDPLATFORM golang:1.20.3 AS deps

ARG TARGETOS TARGETARCH SGCTRL_VER=development
WORKDIR /workspace

COPY . .
# helpful ldflags reference: https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -mod=vendor -ldflags="-X 'main.Version=$SGCTRL_VER' -buildid=" -trimpath -o controller ./cmd/controller/...

# --- run ---
FROM scratch

COPY --from=deps /workspace/controller secretgen-controller

# Run as secretgen-controller by default, will be overridden to a random uid on OpenShift
USER 1000
ENTRYPOINT ["/secretgen-controller"]
