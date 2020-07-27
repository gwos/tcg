const client = require('prom-client');

const GWOS_APP_NAME = 'GW8';
const GWOS_API_TOKEN = '2eb4fc48-d5c6-4086-86ba-1904cd383360';

const registry = new client.Registry();

let gateway = new client.Pushgateway('http://localhost:8099/api/v1', {
    headers: {
        'GWOS-APP-NAME': GWOS_APP_NAME, 'GWOS-API-TOKEN': GWOS_API_TOKEN
    },
}, registry);

const counter = new client.Counter({
    name: 'javascript_counter_example',
    help: 'Counter example in Javascript',
    labelNames: ['critical', 'warning', 'resource', 'group', 'unitType'],
    registers: [registry],
});

let counterInternal = counter.labels(Math.random().toString(), Math.random().toString(), 'Prometheus-Javascript-Push', 'Prometheus-Javascript', 'MB');
counterInternal.inc(Math.random());

const gauge = new client.Gauge({
    name: 'javascript_gauge_example',
    help: 'Gauge example in Javascript',
    labelNames: ['critical', 'warning', 'resource', 'group', 'unitType'],
    registers: [registry],
});

let gaugeInternal = gauge.labels((Math.random() * 100).toString(), (Math.random() * 100).toString(), 'Prometheus-Javascript-Push', 'Prometheus-Javascript', 'MB');
gaugeInternal.set(Math.random() * 100);

gateway.pushAdd({jobName: 'test'}, function (err, resp, body) {
    console.log(body)
});
