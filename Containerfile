FROM registry.access.redhat.com/hi/core-runtime@sha256:e4b8f4ccc7a1ef92c469f53405de43292132f5da648ec5d2f95024f09762f407
COPY customer/message.txt /customer/message.txt
LABEL org.opencontainers.image.source="https://github.com/p5/base-image-provenance-verification"
