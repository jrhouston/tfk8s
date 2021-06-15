FROM golang:1.16-alpine as build
WORKDIR /build
COPY go.* ./
RUN go mod download
COPY . .
RUN apk --no-cache add make
RUN CGO_ENABLED=0 make build

FROM scratch
COPY --from=build /build/tfk8s /bin/tfk8s
ENTRYPOINT ["/bin/tfk8s"]
