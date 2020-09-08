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

const nodes = [
    'node_1',
    'node_2',
];

const labels = [
    'node',
    'service',
    'code',
    'warning',
    'critical',
    'resource',
    'group'
];

// Enable collection of default metrics
require('prom-client').collectDefaultMetrics({
    gcDurationBuckets: [0.001, 0.01, 0.1, 1, 2, 5], // These are the default buckets.
});

// Create custom metrics
const Counter = require('prom-client').Counter;
const c = new Counter({
    name: 'requests_total',
    help: 'Finance Services total http requests made.',
    labelNames: labels,
});

const Gauge = require('prom-client').Gauge;
const g = new Gauge({
    name: 'bytes_transferred',
    help: 'Finance Services bytes transferred over http.',
    labelNames: labels,
});

// Set metric values to some random values for demonstration
for (let i = 0; i < services.length; i++) {
    for (let j = 0; j < nodes.length; j++) {
        g.labels(
            nodes[j],
            services[i],
            '200',
            (Math.random() * (20 - 1) + 1).toFixed(1).toString(),
            (Math.random() * (20 - 1) + 1).toFixed(1).toString(),
            defaultResource,
            defaultGroup,
        ).inc(parseInt((Math.random() * (20 - 1) + 1).toFixed(0)));

        c.labels(
            nodes[j],
            services[i],
            '500',
            (Math.random() * (20 - 1) + 1).toFixed(1).toString(),
            (Math.random() * (20 - 1) + 1).toFixed(1).toString(),
            defaultResource,
            defaultGroup,
        ).inc(parseInt((Math.random() * (20 - 1) + 1).toFixed(0)));
    }
}

// Setup main to Prometheus scrapes:
main.get('/', async (req, res) => {
    try {
        res.set('Content-Type', register.contentType);
        res.end(await response());
    } catch (ex) {
        res.status(500).end(ex);
    }
});

async function response() {
    return await register.getSingleMetricAsString('requests_total') +
        '\n' + await register.getSingleMetricAsString('bytes_transferred') +
        '\n';
}

const port = process.env.PORT || 3000;
console.log(
    `Server listening to :${port}\n`,
);

main.listen(port);
