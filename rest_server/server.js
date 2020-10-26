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

var rabbitMq = amqp.createConnection({ host: 'rabbitmq' })

var hotDBConnection = new Tarantool({
  host: "tarantool",
  port: 3301,
  username: 'tarantool',
  password: 'tarantoolpwd'
});

async function run() {
  await createMongoConnection();
  await createExpressApp();
  await createHTTPServer();
}

async function createExpressApp() {
  expressApp = express();

  expressApp.use(bodyParser.json());

  // TODO verify room id matches existing room

  expressApp.post('/user/create', async (req, res, next) => {
    const id = uuidv4();
    await hotDBConnection.insert('users', {
      "id": id,
      "name": req.body.name
    });

    res.status(200).json({
      "id": id
    });
  });

  expressApp.post('/rooms/create', async (req, res, next) => {
    const id = uuidv4();
    await hotDBConnection.insert('rooms', {
      "id": id,
      "participants": [req.body.uid]
    });
    res.status(200).json({
      "id": id
    });
  });

  expressApp.post('/rooms/:roomId/join', async (req, res, next) => {
    var room = await hotDBConnection.select("rooms", "primary", 1, 0, 'eq', [req.params.roomId]);
    room[0][1].push(req.body.uid);
    await hotDBConnection.update("rooms", "primary", [req.params.roomId], [["=", 1, room[0][1]]]);
    res.status(200).json({});
  });

  expressApp.delete('/rooms/:roomId/leave', async (req, res, next) => {
    var room = await hotDBConnection.select("rooms", "primary", 1, 0, 'eq', [req.params.roomId]);
    const index = room[0][1].indexOf(item);
    room[0][1].splice(index, 1);
    await hotDBConnection.update("rooms", "primary", [req.params.roomId], [["=", 1, room[0][1]]]);
    res.status(200).json({});
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

async function createSocketConn() {
  io = socketIO.listen(expressApp);
  rabbitMq.on('ready', function () {
    io.sockets.on('connection', function (socket) {
       var queue = rabbitMq.queue('my-queue');
 
       queue.bind('#'); // all messages
 
       queue.subscribe(function (message) {
          socket.emit('message-name', message);
       });
    });
 });
}
