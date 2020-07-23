'use strict';

const express = require('express');
const main = express();
const register = require('prom-client').register;

const defaultGroup = 'Prometheus-JS'
const defaultResource = 'javascript-server'

// Enable collection of default metrics

require('prom-client').collectDefaultMetrics({
    gcDurationBuckets: [0.001, 0.01, 0.1, 1, 2, 5], // These are the default buckets.
});

// Create custom metrics

const Histogram = require('prom-client').Histogram;
const h = new Histogram({
    name: 'js_histogram_example',
    help: 'Example of a histogram',
    labelNames: ['critical', 'warning', 'resource', 'unitType', 'group'],
    buckets: [],
});

const Counter = require('prom-client').Counter;
const c = new Counter({
    name: 'js_counter_example',
    help: 'Example of a counter',
    labelNames: ['critical', 'warning', 'resource', 'unitType', 'group'],
});

const Gauge = require('prom-client').Gauge;
const g = new Gauge({
    name: 'js_gauge_example',
    help: 'Example of a gauge',
    labelNames: ['critical', 'warning', 'resource', 'unitType', 'group'],
});

// Set metric values to some random values for demonstration

setTimeout(() => {
    h.labels('200', '500', defaultResource, 'MB', defaultGroup).observe(Math.random());
}, 10);

setInterval(() => {
    c.inc({critical: 150, warning: 330, resource: defaultResource, unitType: 'MB', group: defaultGroup});
}, 5000);

setInterval(() => {
    c.inc({critical: 150, warning: 330, resource: defaultResource, unitType: 'MB', group: defaultGroup});
}, 2000);

setInterval(() => {
    g.set({critical: 150, warning: 330, resource: defaultResource, unitType: 'MB', group: defaultGroup}, Math.random());
    g.labels('150', '350', defaultResource, 'MB', defaultGroup).inc();
}, 100);

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
    return await register.getSingleMetricAsString('js_counter_example') +
        '\n' + await register.getSingleMetricAsString('js_histogram_example') +
        '\n' + await register.getSingleMetricAsString('js_gauge_example') +
        '\n'
}

const port = process.env.PORT || 3000;
console.log(
    `Server listening to :${port}\n`,
);
main.listen(port);
