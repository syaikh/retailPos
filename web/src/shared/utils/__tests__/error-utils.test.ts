import { describe, it, expect } from "vitest";
import { getApiErrorMessage } from "../error-utils";

describe("getApiErrorMessage", () => {
  const fallback = "Something went wrong";

  it("extracts message from response.data.error.message", () => {
    const err = {
      response: { data: { error: { message: "active arrangement exists" } } },
    };
    expect(getApiErrorMessage(err, fallback)).toBe("active arrangement exists");
  });

  it("extracts message when response.data.error is a string", () => {
    const err = { response: { data: { error: "bad request" } } };
    expect(getApiErrorMessage(err, fallback)).toBe("bad request");
  });

  it("falls back to err.message when no response.data.error", () => {
    const err = { message: "timeout" };
    expect(getApiErrorMessage(err, fallback)).toBe("timeout");
  });

  it("returns fallback for null input", () => {
    expect(getApiErrorMessage(null, fallback)).toBe(fallback);
  });

  it("returns fallback for undefined input", () => {
    expect(getApiErrorMessage(undefined, fallback)).toBe(fallback);
  });

  it("returns fallback for primitive input", () => {
    expect(getApiErrorMessage("oops", fallback)).toBe(fallback);
    expect(getApiErrorMessage(42, fallback)).toBe(fallback);
  });

  it("returns fallback for empty object", () => {
    expect(getApiErrorMessage({}, fallback)).toBe(fallback);
  });

  it("returns fallback when error.message is empty string", () => {
    const err = { response: { data: { error: { message: "" } } } };
    expect(getApiErrorMessage(err, fallback)).toBe(fallback);
  });

  it("returns fallback when error.message is undefined", () => {
    const err = { response: { data: { error: {} } } };
    expect(getApiErrorMessage(err, fallback)).toBe(fallback);
  });
});
