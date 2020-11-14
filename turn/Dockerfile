FROM golang:1.15-alpine3.12

ADD . /src/turn

WORKDIR /src/turn
RUN go get
RUN go build

EXPOSE 3478

CMD ["/src/turn/turn", "--users=\"guest=guest\""]

