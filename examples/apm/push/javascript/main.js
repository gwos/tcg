const client = require('prom-client');
const request = require('request');

const GWOS_APP_NAME = '';
let GWOS_API_TOKEN = '';

const registry = new client.Registry();

if (process.argv.length !== 6) {
    console.log('Invalid count of provided arguments. Please check README for details');
    return;
}

// Login
request.post({
    url: process.argv[2] + '/api/auth/login',
    form: {
        user: process.argv[3],
        password: process.argv[4],
        'gwos-app-name': process.argv[5],
    }
}, function (err, httpResponse, body) {
    if (err === null) {
        GWOS_API_TOKEN = body;
    } else {
        console.log('[ERROR]: Couldn\'t login:\n', err);
    }
})

if (GWOS_API_TOKEN === '') {
    return;
}

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

let counterInternal = counter.labels(Math.random().toString(), Math.random().toString(), 'APM-Javascript-Push', 'APM-Javascript', 'MB');
counterInternal.inc(Math.random());

const gauge = new client.Gauge({
    name: 'javascript_gauge_example',
    help: 'Gauge example in Javascript',
    labelNames: ['critical', 'warning', 'resource', 'group', 'unitType'],
    registers: [registry],
});

let gaugeInternal = gauge.labels((Math.random() * 100).toString(), (Math.random() * 100).toString(), 'APM-Javascript-Push', 'APM-Javascript', 'MB');
gaugeInternal.set(Math.random() * 100);

gateway.pushAdd({jobName: 'test'}, function (err, resp, body) {
    console.log(body)
});

// Logout
request.post({
    url: process.argv[2] + '/api/auth/logout',
    form: {
        'gwos-app-name': GWOS_APP_NAME,
        'gwos-api-token': GWOS_API_TOKEN,
    },
    headers: {
        'GWOS-APP-NAME': GWOS_APP_NAME, 'GWOS-API-TOKEN': GWOS_API_TOKEN, 'Content-Type': 'application/x-www-form-urlencoded',
    },
}, function (err, httpResponse, body) {
    console.log("Logout: ", body);
})


