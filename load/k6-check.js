import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";
const scenario = __ENV.SCENARIO || "normal";

const duration = __ENV.DURATION || "5m";
const vus = Number(__ENV.VUS || "3");
const requestSleepSeconds = Number(__ENV.SLEEP || "1");

export const errorRate = new Rate("chainwise_check_errors");
export const checkDuration = new Trend("chainwise_check_duration_ms");

export const options = {
  vus,
  duration,
  thresholds: {
    http_req_failed: ["rate<0.50"],
    http_req_duration: ["p(95)<3000"]
  },
  tags: {
    app: "chainwise",
    endpoint: "frontend_check",
    scenario
  }
};

export default function () {
  const url = `${baseUrl}/check`;

  const response = http.get(url, {
    tags: {
      name: "GET /check",
      scenario
    },
    headers: {
      "X-Load-Scenario": scenario
    }
  });

  const ok = check(response, {
    "status is not 0": (r) => r.status !== 0,
    "response has body": (r) => r.body && r.body.length > 0
  });

  errorRate.add(!ok || response.status >= 500);
  checkDuration.add(response.timings.duration);

  sleep(requestSleepSeconds);
}
