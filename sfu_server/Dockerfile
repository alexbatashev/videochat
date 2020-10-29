FROM golang

ADD . /src/sfu_server

WORKDIR /src/sfu_server
RUN go get
RUN go build

ENTRYPOINT /src/sfu_server/sfu_server
