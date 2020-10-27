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
const amqp = require('amqp');

run();

var expressApp;
var httpServer;
var coldDBConnection;
var socketConn;
var io;

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
    res.setHeader('Access-Control-Allow-Origin', '*');

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
    var ch = await rabbitMQ.createChannel();
    await ch.assertQueue("sfu");
    var msg = {
      "cammand": "create_room",
      "roomId": id,
      "peerId": "",
      "data": ""
    };
    await ch.sendToQueue("sfu", Buffer.from(JSON.stringify(msg)));
    res.status(200).json({
      "id": id
    });
  });

  /*
  expressApp.post('/rooms/:roomId/join', async (req, res, next) => {
  });

  expressApp.delete('/rooms/:roomId/leave', async (req, res, next) => {
    var room = await hotDBConnection.select("rooms", "primary", 1, 0, 'eq', [req.params.roomId]);
    const index = room[0][1].indexOf(item);
    room[0][1].splice(index, 1);
    await hotDBConnection.update("rooms", "primary", [req.params.roomId], [["=", 1, room[0][1]]]);
    res.status(200).json({});
  });
  */
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
  rabbitMQ = await amqp.connect('amqp://rabbitmq');
}

async function createSocketConn() {
  io = socketIO.listen(expressApp);
  io.sockets.on('connection', function (socket) {
    var ch = await rabbitMQ.createChannel();
    await ch.assertQueue("sfu");
    var uid;
    socket.on("join_room", (msg) => {
      var data = JSON.parse(msg);
      var room = await hotDBConnection.select("rooms", "primary", 1, 0, 'eq', [data.roomId]);
      room[0][1].push(data.uid);
      uid = data.uid;
      await hotDBConnection.update("rooms", "primary", [data.roomId], [["=", 1, room[0][1]]]);
    });

    socket.on("leave_room", (msg) => {
      
    });
  });
}
