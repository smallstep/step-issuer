# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM golang:alpine AS builder

RUN mkdir /src
WORKDIR /src
COPY . .

RUN apk add --no-cache make git curl && \
	make CGO_ENABLED=0 V=1 bin/manager

FROM alpine

WORKDIR /
COPY --from=builder /src/bin/manager .
ENTRYPOINT ["/manager"]
