# The builder's Go minor version matches nixpkgs' default, which devenv uses.
FROM docker.io/library/golang:1.26 AS build

WORKDIR /src

# Modules in their own layer, so a source change does not download them again.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Set by the release workflow from the tag; see docs/releasing.md.
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
  -o /harold ./cmd/harold

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /harold /harold

# A TUI: run with a terminal attached, `docker run -it`.
ENTRYPOINT ["/harold"]
