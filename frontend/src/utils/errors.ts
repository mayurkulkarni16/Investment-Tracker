import { isAxiosError } from 'axios';

/** Returns the server's error message for a failed API call, or `fallback`. */
export function apiErrorMessage(err: unknown, fallback: string): string {
  if (isAxiosError<{ error?: string }>(err)) {
    return err.response?.data?.error || fallback;
  }
  return fallback;
}
