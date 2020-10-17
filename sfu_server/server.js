const mediasoup = require('mediasoup');

const config = require('./config');

const mediasoupWorkers = [];

run();

async function run() {
  await runMediasoupWorkers();
}

async function runMediasoupWorkers() {
  const { numWorkers } = config.mediasoup;

	for (let i = 0; i < numWorkers; ++i)
	{
		const worker = await mediasoup.createWorker(
			{
				logLevel   : config.mediasoup.workerSettings.logLevel,
				logTags    : config.mediasoup.workerSettings.logTags,
				rtcMinPort : Number(config.mediasoup.workerSettings.rtcMinPort),
				rtcMaxPort : Number(config.mediasoup.workerSettings.rtcMaxPort)
			});

		worker.on('died', () =>
		{
			setTimeout(() => process.exit(1), 2000);
		});

		mediasoupWorkers.push(worker);

		// Log worker resource usage every X seconds.
		setInterval(async () =>
		{
      const usage = await worker.getResourceUsage();
      // TODO report to pometheus.
		}, 120000);
	}
}
