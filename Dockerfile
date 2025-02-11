FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

WORKDIR /app

# Explicitly download these files before the build so they can be cached
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

ARG TARGETOS TARGETARCH

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go mod tidy && \
    go build -o gleaner

FROM alpine

WORKDIR /app
COPY --from=builder /app/gleaner /app/gleaner
# Update the schema files with the latest version
ADD https://schema.org/version/latest/schemaorg-current-https.jsonld /app/assets/schemaorg-current-https.jsonld
ADD https://schema.org/version/latest/schemaorg-current-http.jsonld /app/assets/schemaorg-current-http.jsonld

ENTRYPOINT ["/app/gleaner"]
