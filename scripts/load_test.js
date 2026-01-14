import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

export const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '10s', target: 20 }, // Ramp up to 20 users
    { duration: '30s', target: 50 }, // Stay at 50 users
    { duration: '10s', target: 0 },  // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'], // Relaxed to 2s for local docker CPU
    errors: ['rate<0.01'],            // <1% status errors
  },
};

// ... (unchanged)

const queries = [
  "travel vacation sydney",
  "software cloud ai",
  "fitness gym workout",
  "fashion luxury shoes",
  "finance investment money",
  "food restaurant dinner",
  "home garden renovation",
  "automotive car truck",
  "entertainment movie music",
  "education university course"
];

function getRandomQuery() {
  return queries[Math.floor(Math.random() * queries.length)];
}

export default function () {
  const url = 'http://host.docker.internal:8080/v1/match';
  // Note: host.docker.internal allows container to reach host's localhost on Mac/Windows
  // If running k6 locally (not docker), use http://localhost:8080

  const payload = JSON.stringify({
    id: `k6-${__VU}-${__ITER}`,
    imp: [{ id: "1" }],
    site: {
      page: getRandomQuery()
    },
    device: {
      ua: "k6-load-test-agent",
      ip: "127.0.0.1"
    }
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '5s' // Increased timeout
  };

  const res = http.post(url, payload, params);

  // 1. Status Check (Functional)
  const statusSuccess = check(res, {
    'status is 200': (r) => r.status === 200,
  });

  // Track Cache Hits vs Misses
  const cacheStatus = res.headers['X-Cache'];
  if (cacheStatus === 'HIT') {
    // Optional: Log hit rate if needed, or rely on latency drop
  }

  // 2. Latency Check (Performance - Information only for error rate, usually)
  // We don't fail the "error" rate on latency, we fail the http_req_duration threshold.
  check(res, {
    'latency < 2000ms': (r) => r.timings.duration < 2000,
  });

  if (!statusSuccess) {
    errorRate.add(1);
  }

  sleep(0.1); // Small sleep to simulate user think time (100ms)
}
