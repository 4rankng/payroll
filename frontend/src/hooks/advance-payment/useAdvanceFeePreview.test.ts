import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useAdvanceFeePreview } from './useAdvanceFeePreview';
import type { CalculateFeeResponse } from '@/types/api/advance-payment.types';

function deferred() {
  let resolve!: (value: CalculateFeeResponse) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<CalculateFeeResponse>((res, rej) => { resolve = res; reject = rej; });
  return { promise, resolve, reject };
}

describe('useAdvanceFeePreview', () => {
  afterEach(() => vi.useRealTimers());

  it('clears the previous fee immediately while the next amount is debounced', async () => {
    vi.useFakeTimers();
    const calculateFee = vi.fn().mockResolvedValue({ fee: 10_000, netAmount: 90_000 });
    const { result } = renderHook(() => useAdvanceFeePreview(calculateFee));
    act(() => result.current.calculate(100_000));
    await act(() => vi.advanceTimersByTimeAsync(300));
    expect(result.current.feeDetails?.netAmount).toBe(90_000);
    act(() => result.current.calculate(200_000));
    expect(result.current.feeDetails).toBeNull();
    expect(calculateFee).toHaveBeenCalledTimes(1);
  });

  it('ignores out-of-order responses for older amounts', async () => {
    vi.useFakeTimers();
    const oldRequest = deferred();
    const currentRequest = deferred();
    const calculateFee = vi.fn().mockReturnValueOnce(oldRequest.promise).mockReturnValueOnce(currentRequest.promise);
    const { result } = renderHook(() => useAdvanceFeePreview(calculateFee));
    act(() => result.current.calculate(100_000));
    await act(() => vi.advanceTimersByTimeAsync(300));
    act(() => result.current.calculate(200_000));
    await act(() => vi.advanceTimersByTimeAsync(300));
    await act(async () => currentRequest.resolve({ fee: 10_000, netAmount: 190_000 }));
    await act(async () => oldRequest.resolve({ fee: 10_000, netAmount: 90_000 }));
    expect(result.current.feeDetails?.netAmount).toBe(190_000);
  });

  it('shows failure and retries the current amount', async () => {
    vi.useFakeTimers();
    const calculateFee = vi.fn().mockRejectedValueOnce(new Error('Network unavailable')).mockResolvedValueOnce({ fee: 10_000, netAmount: 190_000 });
    const { result } = renderHook(() => useAdvanceFeePreview(calculateFee));
    act(() => result.current.calculate(200_000));
    await act(() => vi.advanceTimersByTimeAsync(300));
    expect(result.current.feeError).toBe(true);
    expect(result.current.feeDetails).toBeNull();
    act(() => result.current.retry());
    expect(result.current.feeError).toBe(false);
    await act(() => vi.advanceTimersByTimeAsync(300));
    expect(calculateFee).toHaveBeenLastCalledWith({ amount: 200_000 });
    expect(result.current.feeDetails?.netAmount).toBe(190_000);
  });

  it('does not restore a fee after the input was cleared', async () => {
    vi.useFakeTimers();
    const pending = deferred();
    const { result } = renderHook(() => useAdvanceFeePreview(() => pending.promise));
    act(() => result.current.calculate(200_000));
    await act(() => vi.advanceTimersByTimeAsync(300));
    act(() => result.current.calculate(0));
    await act(async () => pending.resolve({ fee: 10_000, netAmount: 190_000 }));
    expect(result.current.feeDetails).toBeNull();
    expect(result.current.feeError).toBe(false);
  });
});
