process.env.DEBUG = process.env.DEBUG || '*INFO* *WARN* *ERROR*';

// TODO config

const http = require('http');
const bodyParser = require('body-parser');
const express = require('express');
const Tarantool = require('tarantool-driver');
const uuid = require('uuid');
const socketIO = require('socket.io');
const amqp = require('amqplib');
const { exit } = require('process');
const k8s = require('@kubernetes/client-node');
const zookeeper = require('node-zookeeper-client');

const kc = new k8s.KubeConfig();
kc.loadFromCluster();
const k8sApi = kc.makeApiClient(k8s.CoreV1Api);

run();

var expressApp;
var httpServer;
var socketConn;
var io;
var zkClient;

// TODO should this be global?
var ch = null;

var rabbitMQ;

var hotDBConnection = new Tarantool({
  host: "tarantool-routers",
  port: 3301,
  username: 'tarantool',
  password: 'tarantoolpwd'
});

async function run() {
  await createZookeeperConnection();
  await createRabbitMQConnection();
  await createExpressApp();
  await createHTTPServer();
  await createSocketConn();
}

async function createZookeeperConnection() {
  return new Promise(async (resolve, reject) => {
    zkClient = zookeeper.createClient('zookeeper:2181');
    zkClient.once('connected', function () {
      resolve();
    });
    zkClient.connect();
  });
}

async function getNumPeers(path) {
  return new Promise(async (resolve, reject) => {
    zkClient.getData(path, function (error, data, stat) {
      if (error) {
        console.log(error.stack);
        reject();
      }
      console.log(stat);

      const nodeInfo = JSON.parse(data.toString());

      resolve(nodeInfo.NumberOfPeers);
    });
  });
}

async function getBestSFU() {
  return new Promise(async (resolve, reject) => {
    zkClient.getChildren("/sfu", async function (error, children, stats) {
      if (error) {
        console.warn(error);
        reject();
      }
      console.log(stats);

      var minPeers = 1000000;
      var minSfu = "";
      for (const server of children) {
        const numberOfPeers = await getNumPeers("/sfu/" + server);
        console.log(numberOfPeers)
        if (minPeers > numberOfPeers) {
          minPeers = numberOfPeers;
          minSfu = server;
        }
      }

      resolve(minSfu);
    });
  });
}

async function createExpressApp() {
  expressApp = express();

  expressApp.use(bodyParser.urlencoded({ extended: false }));

  expressApp.use(function (req, res, next) {

    // Website you wish to allow to connect
    res.setHeader('Access-Control-Allow-Origin', 'http://localhost');

    // Request methods you wish to allow
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS, PUT, PATCH, DELETE');

    // Request headers you wish to allow
    res.setHeader('Access-Control-Allow-Headers', 'X-Requested-With,content-type');

    // Set to true if you need the website to include cookies in the requests sent
    // to the API (e.g. in case you use sessions)
    res.setHeader('Access-Control-Allow-Credentials', true);

    // Pass to next layer of middleware
    next();
  });

  // TODO verify room id matches existing room

  var router = express.Router()

  router.post('/user/create', async (req, res, next) => {
    const id = uuid.v4();
    console.log(id);
    console.log(req.body.name);
    console.log(req.body);
    try {
      await hotDBConnection.call('addUser', id, req.body.name);
      res.status(200).json({
        "id": id
      });
    } catch (err) {
      console.error(err);
      res.status(500).json({
        "id": null,
        "errMsg": "failed to create user"
      });
    }
  });

  router.post('/rooms/create', async (req, res, next) => {
    const sfuName = await getBestSFU();
    console.log("Best sfu is " + sfuName);
    const id = uuid.v4();
    try {
      await hotDBConnection.call('createRoom', id, sfuName);
    } catch (err) {
      console.error("Failed to create room with id " + id);
      console.error(err);
    }
    ch = await rabbitMQ.createChannel();
    await ch.assertQueue(sfuName, {
      "durable": false,
      "autoDelete": false
    });
    var msg = {
      "command": "create_room",
      "roomId": id,
      "peerId": "",
      "data": ""
    };
    await ch.sendToQueue(sfuName, Buffer.from(JSON.stringify(msg)));
    res.status(200).json({
      "id": id
    });
  });
  router.get("/ice_servers", async (req, res, next) => {
    try {
      const services = await k8sApi.listNamespacedService("default");
      var iceServers = [];
      services.body.items.forEach(service => {
        if (service.metadata["name"] === "turn") {
          const externalIP = service.status.loadBalancer.ingress[0].ip;
          iceServers.push({
            "urls": "stun:" + externalIP + ":3478",
            "username": "guest",
            "credential": "guest"
          });
          iceServers.push({
            "urls": "turn:" + externalIP + ":3478",
            "username": "guest",
            "credential": "guest"
          });
        }
      });
      res.status(200).send(JSON.stringify(iceServers));
    } catch (err) {
      console.error(err);
      res.status(500).send(err);
    }
  });
  expressApp.use('/', router)
}


async function createHTTPServer() {
  httpServer = http.createServer(expressApp);
  await new Promise((resolve, reject) => {
    httpServer.listen(3000, "0.0.0.0", resolve);
  }).catch(() => { });
}

async function createRabbitMQConnection() {
  return new Promise(async function (resolve, reject) {
    const user = process.env.RABBITMQ_USER;
    const pwd = process.env.RABBITMQ_PASSWORD;
    while (true) {
      try {
        rabbitMQ = await amqp.connect('amqp://' + user + ':' + pwd + '@rabbitmq/');
        break;
      } catch (msg) {
        console.warn(msg);
        setTimeout(() => { }, 500);
      }
    }
    resolve();
  });
}

async function createSocketConn() {
  io = socketIO.listen(httpServer);
  io.sockets.on('connection', async (socket) => {
    console.log("New socket connection");
    function setPostHandshakeListeners(socket, sfuChannel, sfuQueue, peerChannel, peerQueue) {
      peerChannel.consume(peerQueue, function (msg) {
        if (msg !== null) {
          const msgString = msg.content.toString();
          var msgObj = JSON.parse(msgString);
          console.log(msgObj)
          if (msgObj.Command === "exchange_offer") {
            console.log(msgObj);
            socket.emit("remote_offer", Buffer.from(JSON.stringify({
              "data": msgObj.Data,
              "kind": msgObj.OfferKind
            })));
            peerChannel.ack(msg);
          } else if (msgObj.Command === "exchange_ice") {
            console.log(msgObj)
            socket.emit("exchange_ice", Buffer.from(
              msgObj.Data
            ));
          }
        }
      });
      socket.on("peer_answer", async (msg) => {
        console.log("Peer answered");
        // TODO send answer to SFU
        var data = JSON.parse(msg)
        try {
          await sfuChannel.sendToQueue(sfuQueue, Buffer.from(JSON.stringify(
            {
              "command": "remote_answer",
              "roomId": data.roomId,
              "peerId": data.uid,
              "data": data.offer
            }
          )));
        } catch (err) {
          console.warn(err);
        }
      });
      socket.on("exchange_ice", async (msg) => {
        console.log("A peer is trying to exchange ICE candidates");
        var data = JSON.parse(msg);
        console.log(data);

        try {
          await sfuChannel.sendToQueue(sfuQueue, Buffer.from(JSON.stringify(
            {
              "command": "exchange_ice",
              "roomId": data.roomId,
              "peerId": data.uid,
              "data": data.ice
            }
          )));
        } catch (err) {
          console.warn(err);
        }
      });
      socket.on("leave_room", async (msg) => {
        console.log("Peer is leaving room");
        var data = JSON.parse(msg);
        await hotDBConnection.call('removeRoomParticipant', data.roomId, data.sessionId);
        await sfuChannel.sendToQueue(sfuQueue, Buffer.from(JSON.stringify(
          {
            "command": "remove_peer",
            "roomId": data.roomId,
            "peerId": data.uid,
            "data": ""
          }
        )));

        // TODO set proper duration
        await hotDBConnection.call('commitSession', data.sessionId, 0);
      });
    }
    socket.on("join_room", async (msg) => {
      console.log("Peer is joining!");
      var data = JSON.parse(msg);

      // This user joins for the first time
      if (data.sessionId === "") {
        const sessionId = uuid.v4();
        try {
          await hotDBConnection.call('startSession', sessionId, Date.now(), data.uid);
        } catch (err) {
          console.error("Failed to create session with id " + sessionId);
          console.error(err);
        }
        try {
          await hotDBConnection.call('addRoomParticipant', data.roomId, sessionId);
        } catch (err) {
          console.error("Failed to add user with session id " + sessionId + " to room with id " + data.roomId);
          console.error(err);
        }

        socket.emit('session_start', JSON.stringify({
          "sessionId": sessionId
        }));
      }

      const room = await hotDBConnection.call('getRoom', data.roomId);
      console.log(room)
      const sfu_worker = room[0][4];
      console.log("SFU worker is " + sfu_worker)
      let sfuChannel = await rabbitMQ.createChannel();
      await sfuChannel.assertQueue(sfu_worker, {
        "durable": false,
        "autoDelete": false
      });
      let peerChannel = await rabbitMQ.createChannel();
      await peerChannel.assertQueue(data.uid, {
        "durable": false,
        "autoDelete": false
      });

      setPostHandshakeListeners(socket, sfuChannel, sfu_worker, peerChannel, data.uid);

      await sfuChannel.sendToQueue(sfu_worker, Buffer.from(JSON.stringify(
        {
          "command": "add_peer",
          "roomId": data.roomId,
          "peerId": data.uid,
          "data": ""
        }
      )));
    });
  });
}
