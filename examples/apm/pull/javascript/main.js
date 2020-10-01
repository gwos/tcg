'use strict';

const express = require('express');
const main = express();
const register = require('prom-client').register;

const defaultResource = 'FinanceServicesJS';
const defaultGroup = 'PrometheusDemo';

const services = [
    'analytics',
    'distribution',
    'sales',
];

const labels = [
    'service',
    'warning',
    'critical',
    'resource',
    'group'
];

// Enable collection of default metrics
require('prom-client').collectDefaultMetrics({
    gcDurationBuckets: [0.001, 0.01, 0.1, 1, 2, 5],
});

// Create custom metrics
const Counter = require('prom-client').Counter;
const requestsPerMinute = new Counter({
    name: 'requests_per_minute',
    help: 'Finance Services http requests per minute.',
    labelNames: labels,
});

const Gauge = require('prom-client').Gauge;
const bytesPerMinute = new Gauge({
    name: 'bytes_per_minute',
    help: 'Finance Services bytes transferred over http per minute.',
    labelNames: labels,
});

const responseTime = new Gauge({
    name: 'response_time',
    help: 'Finance Services http response time average over 1 minute.',
    labelNames: labels,
});

// Set metric values to some random values for demonstration:
handleMetrics();

// Simulate request traffic:
setInterval(handleMetrics, 30000);

// Setup main to Prometheus scrapes:
main.get('/metrics', async (req, res) => {
    try {
        res.set('Content-Type', register.contentType);
        res.end(await response());
    } catch (ex) {
        res.status(500).end(ex);
    }
});

main.get('/', async (req, res) => {
    try {
        res.end('Groundwork Prometheus Metrics example. Hit the /metrics end point to see Prometheus Exposition metrics...');
    } catch (ex) {
        res.status(500).end(ex);
    }
});

async function response() {
    return await register.getSingleMetricAsString('requests_per_minute') +
        '\n' + await register.getSingleMetricAsString('bytes_per_minute') +
        '\n' + await register.getSingleMetricAsString('response_time') +
        '\n';
}

const port = process.env.PORT || 3000;
console.log(
    `Server listening to :${port}\n`,
);

main.listen(port);

function handleMetrics() {
    bytesPerMinute.reset();
    requestsPerMinute.reset();
    responseTime.reset();

    for (let i = 0; i < services.length; i++) {
        bytesPerMinute.labels(
            services[i],
            '40000',
            '45000',
            defaultResource,
            defaultGroup,
        ).inc(parseInt((Math.random() * (60000 - 10000) + 10000).toFixed(0)));

        requestsPerMinute.labels(
            services[i],
            '70',
            '90',
            defaultResource,
            defaultGroup,
        ).inc(parseInt((Math.random() * (100 - 1) + 1).toFixed(0)));

        responseTime.labels(
            services[i],
            '2.0',
            '2.5',
            defaultResource,
            defaultGroup,
        ).inc(parseFloat((Math.random() * (3)).toFixed(1)));
    }
}
