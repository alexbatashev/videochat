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

run();

var expressApp;
var httpServer;
var coldDBConnection;

var hodDBConnection = new Tarantool({
  host: "tarantool",
  port: 3301,
  username: 'tarantool',
  password: 'tarantoolpwd'
});

async function run() {
  console.log("Boom");
  await createExpressApp();
  await createHTTPServer();
  await createMongoConnection();
}

async function createExpressApp() {
  expressApp = express();

  expressApp.use(bodyParser.json());

  // TODO verify room id matches existing room

  expressApp.post('/rooms/create', async (req, res, next) => {
    res.status(200).json({});
  });

  expressApp.post('/rooms/:roomId/join', async (req, res, next) => {
    res.status(200).json({});
  });

  expressApp.delete('/rooms/:roomId/leave', async (req, res, next) => {
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
