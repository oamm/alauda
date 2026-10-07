import "@testing-library/jest-dom/vitest";

// jsdom has no layout engine; Radix form controls observe their native-input size.
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};
