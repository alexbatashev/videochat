FROM clearlinux/golang:1.15 as builder

ADD . /src/turn

WORKDIR /src/turn
RUN go get
RUN go build

EXPOSE 3478

FROM clearlinux:latest

WORKDIR /dist

# Copy the build artifacts from the previous stage
COPY --from=builder /src/turn/turn .

# Run the executable
CMD ["./turn", "--users=\"guest=guest\""]
