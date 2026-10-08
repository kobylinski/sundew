FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
COPY internal/ui/static/placeholder.html /src/internal/ui/static/placeholder.html
COPY prototypes/web-ui/src/style.css /src/prototypes/web-ui/src/style.css
RUN npm test

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/ui/static/ ./internal/ui/static/
ARG TARGETOS TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o /out/sundew ./cmd/sundew

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/sundew /sundew
EXPOSE 8025
ENTRYPOINT ["/sundew"]
