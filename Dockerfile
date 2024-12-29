ARG GO_VERSION=1.22.10

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build
COPY . .

# fuck google
RUN go env -w GOPROXY=direct

RUN go install github.com/a-h/templ/cmd/templ@latest
RUN templ generate

RUN go install github.com/dlclark/regexp2cg@main
RUN go generate ./lib/*

RUN go install git.maid.zone/stuff/soundcloakctl@master
RUN soundcloakctl config codegen
# this downloads JS modules (currently only hls.js) from jsdelivr and stores them locally for serving
RUN soundcloakctl js download 

RUN CGO_ENABLED=0 GOARCH=${TARGETARCH} GOOS=${TARGETOS} go build -ldflags "-s -w -extldflags '-static'" -o ./app
RUN echo "soundcloak:x:5000:5000:Soundcloak user:/:/sbin/nologin" > /etc/minimal-passwd && \
  echo "soundcloak:x:5000:" > /etc/minimal-group

RUN soundcloakctl postbuild
  
FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build/static/assets /static/assets
COPY --from=build /build/static/instance /static/instance
COPY --from=build /build/static/external /static/external
COPY --from=build /build/app /app
COPY --from=build /etc/minimal-passwd /etc/passwd
COPY --from=build /etc/minimal-group /etc/group

EXPOSE 4664

USER soundcloak

ENTRYPOINT ["/app"]
