process.title = 'mediasoup-demo-server';
process.env.DEBUG = process.env.DEBUG || '*INFO* *WARN* *ERROR*';

// TODO config

const fs = require('fs');
const http = require('http');
const url = require('url');
const bodyParser = require('body-parser');
const express = require('express');
const Tarantool = require('tarantool-driver');
const MongoClient = require('mongodb').MongoClient;
const { v4: uuidv4 } = require('uuid');
const socketIO = require('socket.io');
const amqp = require('amqplib');
const { exit } = require('process');

run();

var expressApp;
var httpServer;
var coldDBConnection;
var socketConn;
var io;

// TODO should this be global?
var ch = null;

var rabbitMQ;

var hotDBConnection = new Tarantool({
  host: "tarantool",
  port: 3301,
  username: 'tarantool',
  password: 'tarantoolpwd'
});

async function run() {
  await createRabbitMQConnection();
  await createMongoConnection();
  await createExpressApp();
  await createHTTPServer();
  await createSocketConn();
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

  expressApp.post('/user/create', async (req, res, next) => {
    const id = uuidv4();
    console.log(id);
    console.log(req.body.name);
    console.log(req.body);
    await hotDBConnection.insert('users', [
      id,
      req.body.name
    ]);

    res.status(200).json({
      "id": id
    });
  });

  expressApp.post('/rooms/create', async (req, res, next) => {
    const id = uuidv4();
    await hotDBConnection.insert('rooms', [id, []]);
    ch = await rabbitMQ.createChannel();
    await ch.assertQueue("sfu", {
      "durable": false
    });
    var msg = {
      "command": "create_room",
      "roomId": id,
      "peerId": "",
      "data": ""
    };
    await ch.sendToQueue("sfu", Buffer.from(JSON.stringify(msg)));
    res.status(200).json({
      "id": id
    });
  });
}

async function createHTTPServer() {
  httpServer = http.createServer(expressApp);
  await new Promise((resolve, reject) => {
    httpServer.listen(3000, "0.0.0.0", resolve);
  }).catch(() => { });
}

async function createMongoConnection() {
  MongoClient.connect("mongodb://mongo:27017", function (_, client) {
    coldDBConnection = client;
  });
}

async function createRabbitMQConnection() {
  try {
    rabbitMQ = await amqp.connect('amqp://guest:guest@rabbitmq/');
  } catch (msg) {
    console.warn(msg);
    exit(1);
  }
}

async function createSocketConn() {
  io = socketIO.listen(httpServer);
  io.sockets.on('connection', async function (socket) {
    console.log("New socket connection");
    var ch = null;
    socket.on("handshake", async (msg) => {
      var data = JSON.parse(msg);
      console.log("Handshake");
      console.log(data);
        console.log("Creating channel")
        ch = await rabbitMQ.createChannel();
        await ch.assertQueue("sfu", {
          "durable": false
        });
        console.log("Created channel connection");
    });
    socket.on("join_room", async (msg) => {
      console.log("A peer is trying to join room");
      var data = JSON.parse(msg);
      console.log(data);
      var room = await hotDBConnection.select("rooms", "primary", 1, 0, 'eq', [data.roomId]);
      room[0][1].push(data.uid);

      await hotDBConnection.update("rooms", "primary", [data.roomId], [["=", 1, room[0][1]]]);
      console.log("Updated db");

      try {
        var peerChannel = await rabbitMQ.createChannel();
        await peerChannel.assertQueue(data.uid, {
          "durable": false
        });

        peerChannel.consume(data.uid, function (msg) {
          console.log("Accept message from " + data.uid);
          if (msg !== null) {
            var msgObj = JSON.parse(msg.content.toString());
            console.log(msgObj)
            if (msgObj.Command === "exchange_offer") {
              console.log(msgObj);
              socket.emit("remote_offer", Buffer.from(
                msgObj.Data
              ));
              peerChannel.ack(msg);
            } else if (msgObj.Command === "exchange_ice") {
              console.log(msgObj)
              socket.emit("exchange_ice", Buffer.from(
                msgObj.Data
              ));
            }
          }
        });

        await ch.sendToQueue("sfu", Buffer.from(JSON.stringify(
          {
            "command": "add_peer",
            "roomId": data.roomId,
            "peerId": data.uid,
            "data": data.offer
          }
        )));
        console.log("Sent request to SFU");
      } catch (err) {
        console.warn(err);
      }
    });

    socket.on("exchange_ice", async (msg) => {
      console.log("A peer is trying to exchange ICE candidates");
      var data = JSON.parse(msg);
      console.log(data);

      try {
        await ch.sendToQueue("sfu", Buffer.from(JSON.stringify(
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

    socket.on("leave_room", (msg) => {

    });
  });
}
