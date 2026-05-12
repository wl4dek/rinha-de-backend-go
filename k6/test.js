import http from 'k6/http';
import { SharedArray } from 'k6/data';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

const errors = [];

const payloads = new SharedArray('payloads', function () {
    return JSON.parse(open('/scripts/payloads.json'));
});

export const options = {
    stages: [
        { duration: '30s', target: 50, tags: { stage: 'warmup' } },
        { duration: '1m', target: 300, tags: { stage: 'load' } },
        { duration: '2m', target: 800, tags: { stage: 'stress' } },
        { duration: '30s', target: 0, tags: { stage: 'cooldown' } },
    ],

    thresholds: {
        http_req_failed: ['rate<0.01'],
        http_req_duration: ['p(95)<500'],
    },
};

export default function () {
    const payload =
        payloads[Math.floor(Math.random() * payloads.length)];

    const res = http.post(
        'http://nginx:9999/fraud-score',
        JSON.stringify(payload),
        {
            headers: {
                'Content-Type': 'application/json',
            },
        }
    );

    if (res.status !== 200) {
        errors.push({
            status: res.status,
            body: res.body,
        });
    }
}

export function handleSummary(data) {
    return {
        '/scripts/errors.json': JSON.stringify(errors, null, 2),
        '/scripts/report.html': htmlReport(data),
    };
}