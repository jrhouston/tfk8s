FROM golang:1.15-buster as build
WORKDIR /build
ARG version
COPY go.* .
RUN go mod download
COPY . .
ENV CGO_ENABLED 0
RUN  go build -ldflags "-X main.toolVersion=$version" -o tfk8s && \
     chmod +x tfk8s

FROM alpine
COPY --from=build /build/tfk8s /bin/tfk8s
ENTRYPOINT ["/bin/tfk8s"]

