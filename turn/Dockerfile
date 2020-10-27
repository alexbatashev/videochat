FROM golang

ADD . /src/turn

WORKDIR /src/turn
RUN go get
RUN go build

CMD ["/src/turn/turn", "--users", "guest=guest", "--realm", "localhost"]

