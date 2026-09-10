interface AxiosLikeError {
  response?: {
    data?: {
      error?: string | { message?: string };
    };
  };
  message?: string;
}

export function getApiErrorMessage(
  e: unknown,
  fallback: string,
): string {
  if (!e || typeof e !== "object") return fallback;
  const err = e as AxiosLikeError;
  const apiError = err.response?.data?.error;
  if (apiError) {
    if (typeof apiError === "string") return apiError;
    if (apiError.message) return apiError.message;
  }
  if (err.message) return err.message;
  return fallback;
}
