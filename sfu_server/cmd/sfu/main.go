package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/alexbatashev/videochat/pkg/sfu"
)

func main() {
	forever := make(chan bool)

	rc := sfu.CreateWebRTCRoomController()
	qp, err := sfu.CreateRabbitMQProvider("amqp://guest:guest@rabbitmq/")
	if err != nil {
		panic(err)
	}

	go sfu.StartServer(qp, "sfu", &rc)

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

	<-forever
}
