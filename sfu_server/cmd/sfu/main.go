package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexbatashev/videochat/pkg/sfu"
	"github.com/go-zookeeper/zk"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type NodeInfo struct {
	NumberOfPeers uint32
}

func main() {
	forever := make(chan bool)

	podName := os.Getenv("SFU_POD_NAME")
	if podName == "" {
		podName = "sfu"
	}

	log.Printf("Pod name is %s", podName)

	time.Sleep(3*time.Second)
	zkClient, done, err := zk.Connect([]string{"zookeeper"}, time.Second*10)
	if err != nil {
		panic(err)
	}

	for true {
		e := <-done
		if e.State == zk.StateConnected {
			break
		}
		time.Sleep(30*time.Millisecond)
	}
	defer zkClient.Close()

	exists, _, err := zkClient.Exists("/sfu")
	if !exists && err == nil {
		zkClient.Create("/sfu", nil, 0, zk.WorldACL(zk.PermAll))
	} else if err != nil {
		panic(err)
	}

	nodeInfo := NodeInfo{0}
	nodeInfoJSON, err := json.Marshal(nodeInfo)
	if err != nil {
		panic(err)
	}

	_, err = zkClient.Create("/sfu/"+podName, nodeInfoJSON, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		panic(err)
	}

	rc := sfu.CreateWebRTCRoomController()
	// user := os.Getenv("RABBITMQ_USER") 
	// password := os.Getenv("RABBITMQ_PASSWORD") 	
	// url := "amqp://" + user + ":" + password + "@rabbitmq/"
	qp, err := sfu.CreateKafkaProvider("kafka:9092")
	if err != nil {
		panic(err)
	}
	defer qp.Close()

	var numberOfPeers uint32 = 0
	var version int32 = 0

	go sfu.StartServer(qp, podName, &rc, func (direction bool) {
		if direction {
			numberOfPeers += 1
		} else {
			numberOfPeers -= 1
		}
		log.Printf("Number of peers is %d", numberOfPeers)
		nodeInfo := NodeInfo{numberOfPeers}
		nodeInfoJSON, err := json.Marshal(nodeInfo)
		if err != nil {
			panic(err)
		}
		_, err = zkClient.Set("/sfu/" + podName, nodeInfoJSON, version)
		if err != nil {
			log.Print(err)
		}
		version += 1
	})

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

	<-forever
}
