FROM alpine
COPY ./initializer/multiarch-initializer /multiarch-initializer
ENTRYPOINT ["/multiarch-initializer"]
