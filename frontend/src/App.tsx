import { FormEvent, useEffect, useRef, useState } from "react";
import { checkPassword } from "./pwned";

type State =
  | { kind: "idle" }
  | { kind: "loading" }
  | { kind: "result"; count: number; prefix: string }
  | { kind: "error"; message: string };

const fmt = new Intl.NumberFormat();

export default function App() {
  const [password, setPassword] = useState("");
  const [reveal, setReveal] = useState(false);
  const [state, setState] = useState<State>({ kind: "idle" });
  const [total, setTotal] = useState<number | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const loadStats = () =>
    fetch("/api/stats", { cache: "no-store" })
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => j && setTotal(j.totalChecks))
      .catch(() => {});

  useEffect(() => {
    loadStats();
  }, []);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!password) return;
    abortRef.current?.abort();
    const ac = new AbortController();
    abortRef.current = ac;
    setState({ kind: "loading" });
    try {
      const r = await checkPassword(password, ac.signal);
      setState({ kind: "result", ...r });
      loadStats();
    } catch (err) {
      if ((err as Error).name === "AbortError") return;
      setState({ kind: "error", message: (err as Error).message || "Something went wrong" });
    }
  }

  function onChange(v: string) {
    setPassword(v);
    if (state.kind !== "loading") setState({ kind: "idle" });
  }

  return (
    <div className="page">
      <header className="masthead">
        <div className="mark" aria-hidden>
          <span />
        </div>
        <span className="brand">LeakCheck</span>
        <span className="tag">free · no accounts · no tracking</span>
      </header>

      <main>
        <h1>
          Has your password <em>leaked?</em>
        </h1>
        <p className="lede">
          Check it against more than a billion passwords exposed in real data breaches. Your password is hashed on
          this device — it never leaves your browser.
        </p>

        <form className="checker" onSubmit={onSubmit} autoComplete="off">
          <label htmlFor="pw" className="sr-only">
            Password to check
          </label>
          <div className="field">
            <input
              id="pw"
              type={reveal ? "text" : "password"}
              value={password}
              onChange={(e) => onChange(e.target.value)}
              placeholder="Type a password"
              autoComplete="off"
              autoCorrect="off"
              autoCapitalize="off"
              spellCheck={false}
              data-lpignore="true"
              data-1p-ignore
              autoFocus
            />
            <button type="button" className="ghost" onClick={() => setReveal((r) => !r)} aria-pressed={reveal}>
              {reveal ? "Hide" : "Show"}
            </button>
          </div>
          <button type="submit" className="primary" disabled={!password || state.kind === "loading"}>
            {state.kind === "loading" ? "Checking…" : "Check password"}
          </button>
        </form>

        <div aria-live="polite">
          {state.kind === "result" && state.count > 0 && (
            <section className="result bad">
              <div className="big">{fmt.format(state.count)}</div>
              <div>
                <h2>Oh no — this password has been leaked.</h2>
                <p>
                  It appears {state.count === 1 ? "once" : `${fmt.format(state.count)} times`} in known data breaches.
                  Attackers try these first. If you use it anywhere, change it there now — and don’t reuse it.
                </p>
              </div>
            </section>
          )}
          {state.kind === "result" && state.count === 0 && (
            <section className="result good">
              <div className="big">0</div>
              <div>
                <h2>Good news — not found in any known breach.</h2>
                <p>
                  That doesn’t make it strong. Use a long, unique password for every site, ideally from a password
                  manager.
                </p>
              </div>
            </section>
          )}
          {state.kind === "error" && (
            <section className="result err">
              <div>
                <h2>Couldn’t check right now</h2>
                <p>{state.message}</p>
              </div>
            </section>
          )}
          {state.kind === "result" && (
            <p className="sent">
              Sent over the network: <code>{state.prefix}</code> — the first 5 of 40 characters of the hash.
            </p>
          )}
        </div>

        <section className="how">
          <h3>How your password stays private</h3>
          <ol>
            <li>
              <b>Hashed locally.</b> Your browser computes the SHA-1 hash of what you typed, using the built-in Web
              Crypto API.
            </li>
            <li>
              <b>Only 5 characters go out.</b> Just the first 5 hex characters of that hash are sent. Hundreds of
              different leaked passwords share any given prefix.
            </li>
            <li>
              <b>Matched on your device.</b> The server returns every leaked hash suffix with that prefix (padded with
              decoys), and your browser looks for a match itself.
            </li>
          </ol>
          <p className="fine">
            Breach data from{" "}
            <a href="https://haveibeenpwned.com/Passwords" target="_blank" rel="noreferrer">
              Have I Been Pwned
            </a>{" "}
            Pwned Passwords. We store no passwords, hashes, or IP addresses — only a cached copy of public range
            responses and a running count of checks.
          </p>
        </section>
      </main>

      <footer>
        {total !== null && <span>{fmt.format(total)} checks run</span>}
        <span>Free forever · open source</span>
      </footer>
    </div>
  );
}
