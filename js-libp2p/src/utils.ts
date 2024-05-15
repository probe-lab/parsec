/**
 * Convert ms to s
 */
export function toSeconds (ms: number): number {
  return ms / 1000
}

/**
 * Convert ms to ns
 */
export function toNanoSeconds (ms: number): number {
  return ms * 1000000
}
