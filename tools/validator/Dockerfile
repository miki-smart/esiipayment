# Builds the esiipayment CLI as a single static binary, then ships it in a
# minimal runtime image so no contributor needs a Go toolchain installed
# to run `esiipayment validate` / `esiipayment replay` / `esiipayment lint`. See
# docs/add-a-provider.md.

FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY internal ./internal
COPY cmd ./cmd
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /out/esiipayment ./cmd/esiipayment

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/esiipayment /usr/local/bin/esiipayment
ENTRYPOINT ["/usr/local/bin/esiipayment"]
