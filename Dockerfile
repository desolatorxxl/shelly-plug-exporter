FROM golang:1.18 as build

COPY . /go/shelly-plug-exporter

WORKDIR /go/shelly-plug-exporter

RUN CGO_ENABLED=0 go build && ls

FROM alpine

COPY --from=build /go/shelly-plug-exporter/shelly-plug-exporter /usr/local/bin/shelly-plug-exporter

USER 9956

ENTRYPOINT ["shelly-plug-exporter"]
