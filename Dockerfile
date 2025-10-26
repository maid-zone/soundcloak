ARG GO_VERSION=1.25.3

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
RUN go mod download -x
COPY . .
RUN go tool soundcloakctl js download && \
  go tool templ generate && \
  go generate ./lib/* && \
  go tool soundcloakctl config codegen && \
  go tool soundcloakctl -nozstd precompress && \
  CGO_ENABLED=0 GOARCH=${TARGETARCH} GOOS=${TARGETOS} go build -v -ldflags "-s -w -extldflags '-static' -X git.maid.zone/stuff/soundcloak/lib/cfg.Commit=`git rev-parse --short HEAD` -X git.maid.zone/stuff/soundcloak/lib/cfg.Repo=`git remote get-url origin`" -o ./app && \
  go tool soundcloakctl postbuild
  
FROM scratch

COPY --from=build /build/static/ /static/
COPY --from=build /build/app /app
COPY --from=build /etc2/ /etc/

EXPOSE 4664

USER soundcloak

ENTRYPOINT ["/app"]
