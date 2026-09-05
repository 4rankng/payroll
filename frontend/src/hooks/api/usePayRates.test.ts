import { describe, expect, it } from 'vitest';
import { payrateQueryRetry, isForbiddenError } from './usePayRates';

const axiosErr = (status: number) => ({ response: { status } });
const apiErr = (status: number) => ({ http_status: status });

describe('payrateQueryRetry', () => {
  it('never retries 404 — no payrate exists for the project', () => {
    expect(payrateQueryRetry(0, axiosErr(404))).toBe(false);
    expect(payrateQueryRetry(2, apiErr(404))).toBe(false);
  });

  it('never retries 403 — no project access; retrying cannot change the answer', () => {
    // Regression: 403 used to be retried 3x, turning one forbidden payrate
    // fetch into 4 error rows (prod burst pattern: 4 identical 403s/minute).
    expect(payrateQueryRetry(0, axiosErr(403))).toBe(false);
    expect(payrateQueryRetry(2, apiErr(403))).toBe(false);
  });

  it('retries transient errors up to 3 times', () => {
    expect(payrateQueryRetry(0, axiosErr(500))).toBe(true);
    expect(payrateQueryRetry(2, apiErr(502))).toBe(true);
    expect(payrateQueryRetry(3, axiosErr(500))).toBe(false);
    expect(payrateQueryRetry(1, new Error('network'))).toBe(true);
  });
});

describe('isForbiddenError', () => {
  it('matches both axios and API error shapes for 403 only', () => {
    expect(isForbiddenError(axiosErr(403))).toBe(true);
    expect(isForbiddenError(apiErr(403))).toBe(true);
    expect(isForbiddenError(axiosErr(404))).toBe(false);
    expect(isForbiddenError(apiErr(500))).toBe(false);
    expect(isForbiddenError(null)).toBe(false);
    expect(isForbiddenError('not an error')).toBe(false);
  });
});
