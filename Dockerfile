ARG GO_VERSION=1.21.3
ARG NODE_VERSION=bookworm

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
ARG TARGETOS
ARG TARGETARCH

WORKDIR /build
COPY . .
RUN go install github.com/a-h/templ/cmd/templ@latest && \
  templ generate && \
  CGO_ENABLED=0 GOARCH=${TARGETARCH} GOOS=${TARGETOS} go build -ldflags "-s -w -extldflags '-static'" -o ./app && \
  echo "soundcloak:x:5000:5000:Soundcloak user:/:/sbin/nologin" > /etc/minimal-passwd && \
  echo "soundcloak:x:5000:" > /etc/minimal-group

FROM node:${NODE_VERSION} AS node
WORKDIR /hls.js
COPY --from=build /build/package.json ./
RUN npm i

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build/assets /assets
COPY --from=build /build/app /app
COPY --from=node /hls.js/node_modules/hls.js/dist/hls.light.min.js /node_modules/hls.js/dist/hls.light.min.js
COPY --from=build /etc/minimal-passwd /etc/passwd
COPY --from=build /etc/minimal-group /etc/group

EXPOSE 4664

USER soundcloak

ENTRYPOINT ["/app"]
