// All hashing happens here, in the browser. Only the first 5 hex characters of
// the SHA-1 hash are ever sent over the network (HIBP k-anonymity model).

export async function sha1Hex(input: string): Promise<string> {
  const data = new TextEncoder().encode(input);
  const digest = await crypto.subtle.digest("SHA-1", data);
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("")
    .toUpperCase();
}

/** Find the breach count for `suffix` in a range response body. Padding rows (count 0) are ignored. */
export function findCount(body: string, suffix: string): number {
  for (const line of body.split(/\r?\n/)) {
    const idx = line.indexOf(":");
    if (idx !== 35) continue;
    if (line.slice(0, 35).toUpperCase() === suffix) {
      const n = parseInt(line.slice(36), 10);
      return Number.isFinite(n) ? n : 0;
    }
  }
  return 0;
}

export interface CheckResult {
  count: number;
  prefix: string;
}

export async function checkPassword(password: string, signal?: AbortSignal): Promise<CheckResult> {
  const hash = await sha1Hex(password);
  const prefix = hash.slice(0, 5);
  const suffix = hash.slice(5);
  const res = await fetch(`/api/range/${prefix}`, { signal, cache: "no-store" });
  if (!res.ok) {
    let msg = `Lookup failed (${res.status})`;
    try {
      const j = await res.json();
      if (j?.error) msg = j.error;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }
  return { count: findCount(await res.text(), suffix), prefix };
}
