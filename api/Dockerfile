# build stage
FROM golang:alpine AS build-stage
WORKDIR /go/src/github.com/dairycart/dairycart

ADD . .
RUN go build -o /dairycart
COPY migrations/ /migrations

# final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates

COPY --from=build-stage /dairycart /dairycart
COPY --from=build-stage /migrations /migrations

ENV DAIRYSECRET "do-not-use-secrets-like-this-plz"
ENV MIGRATE_EXAMPLE_DATA $MIGRATE_EXAMPLE_DATA
ENV DAIRYCART_MIGRATIONS_DIR 'file://migrations'

ENTRYPOINT ["/dairycart"]
