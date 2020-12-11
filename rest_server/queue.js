const { Kafka, logLevel, CompressionTypes } = require('kafkajs');
const PrettyConsoleLogger = require('./prettyConsoleLogger.js')

class Queue {
  constructor(name, producer, consumer) {
    this.producer = producer;
    this.consumer = consumer;
    this.topic = name;
    this.receive_cb = async (msg) => { }
  }

  async send(message) {
    await this.producer.send({
      topic: this.topic,
      messages: [{
        key: null,
        value: Buffer.from(JSON.stringify(message)).toString('base64')
      }]
    });
    console.log("Sending " + JSON.stringify(message));
  }

  async receive(cb) {
    this.receive_cb = cb;
    await this.consumer.connect();
    await this.consumer.subscribe({ topic: this.topic, fromBeginning: true });
    await this.consumer.run({
      eachMessage: async ({ topic, partition, message }) => {
        console.log("Received a message")
        console.log(message)
        if (topic === this.topic) {
          await cb(message.value);
        }
      }
    });
  }

  async _receive_cb(msg) {
    await this.receive_cb(msg);
  }
}

class QueueProvider {
  constructor(url, clientId) {
    this.kafka = new Kafka({
      logLevel: logLevel.INFO,
      logCreator: PrettyConsoleLogger,
      brokers: [url],
      clientId: clientId,
      connectionTimeout: 3000,
      retry: {
        initialRetryTime: 100,
        retries: 800
      }
    });
    this.clientId = clientId;
    this.producer = this.kafka.producer();
  }

  async init() {
    await this.producer.connect();
  }

  async createQueue(name) {
    const consumer = this.kafka.consumer({ groupId: this.clientId + "-" + name });
    const queue = new Queue(name, this.producer, consumer);
    return queue;
  }

  async assertQueue(name) {
    const admin = this.kafka.admin();
    await admin.connect();
    await admin.createTopics({
      validateOnly: false,
      waitForLeaders: true,
      timeout: 300,
      topics: [{
        topic: name,
        replicationFactor: 3
      }]
    });
    await admin.disconnect();
  }
}

module.exports = { Queue, QueueProvider };
