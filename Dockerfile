ARG GO_VERSION=1.25.0

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

RUN go env -w GOPROXY=direct && \
  mkdir /etc2 && \
  mkdir /etc2/ssl && mkdir /etc2/ssl/certs && \
  cp /etc/ssl/certs/ca-certificates.crt /etc2/ssl/certs/ca-certificates.crt && \
  echo "soundcloak:x:5000:5000:Soundcloak user:/:/sbin/nologin" > /etc2/passwd && \
  echo "soundcloak:x:5000:" > /etc2/group

COPY go.* .
RUN go mod download -x && \
  go install -v github.com/a-h/templ/cmd/templ@latest && \
  go install -v github.com/dlclark/regexp2cg@main
COPY . .
# usually soundcloakctl updates together with soundcloak, so we should redownload it
RUN go install -v git.maid.zone/stuff/soundcloakctl@master && \
  soundcloakctl js download && \
  templ generate && \
  go generate ./lib/* && \
  soundcloakctl config codegen && \
  soundcloakctl -nozstd precompress && \
  CGO_ENABLED=0 GOARCH=${TARGETARCH} GOOS=${TARGETOS} go build -v -ldflags "-s -w -extldflags '-static' -X git.maid.zone/stuff/soundcloak/lib/cfg.Commit=`git rev-parse HEAD | head -c 7` -X git.maid.zone/stuff/soundcloak/lib/cfg.Repo=`git remote get-url origin`" -o ./app && \
  soundcloakctl postbuild
  
FROM scratch

COPY --from=build /build/static/ /static/
COPY --from=build /build/app /app
COPY --from=build /etc2/ /etc/

EXPOSE 4664

USER soundcloak

ENTRYPOINT ["/app"]
