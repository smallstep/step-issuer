# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:latest
ARG BINPATH="docker/bin/manager"
WORKDIR /
COPY $BINPATH .
ENTRYPOINT ["/manager"]
