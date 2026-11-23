import { FormEvent, useEffect, useRef, useState } from "react";
import { gsap } from "gsap";
import { useGSAP } from "@gsap/react";
import { checkPassword } from "./pwned";
import { prefersReducedMotion, useLenis } from "./useLenis";
import SplitText from "./components/reactbits/SplitText";
import ShinyText from "./components/reactbits/ShinyText";
import SpotlightCard from "./components/reactbits/SpotlightCard";
import SoftAurora from "./components/reactbits/SoftAurora";
import AnimatedContent from "./components/reactbits/AnimatedContent";

gsap.registerPlugin(useGSAP);

const fmt = new Intl.NumberFormat();

type State =
  | { kind: "idle" }
  | { kind: "loading" }
  | { kind: "result"; count: number; prefix: string }
  | { kind: "error"; message: string };

const STEPS = [
  {
    n: "01",
    title: "Hashed on your device",
    body: "Your browser turns the password into a SHA-1 hash using the built-in Web Crypto API. Nothing is sent yet.",
  },
  {
    n: "02",
    title: "Only 5 characters leave",
    body: "Just the first 5 of the hash’s 40 characters are sent. Hundreds of leaked passwords share every prefix.",
  },
  {
    n: "03",
    title: "Matched in your browser",
    body: "We return every leaked hash with that prefix, padded with decoys, and your browser looks for a match itself.",
  },
];

function ResultCard({ count, prefix }: { count: number; prefix: string }) {
  const root = useRef<HTMLDivElement>(null);
  const num = useRef<HTMLSpanElement>(null);
  const leaked = count > 0;

  useGSAP(
    () => {
      const reduce = prefersReducedMotion();
      gsap.from(root.current, { y: reduce ? 0 : 18, opacity: 0, scale: reduce ? 1 : 0.98, duration: 0.6, ease: "power3.out" });
      if (!num.current) return;
      if (reduce || !leaked) {
        num.current.textContent = fmt.format(count);
        return;
      }
      const o = { v: 0 };
      gsap.to(o, {
        v: count,
        duration: Math.min(1.8, 0.6 + Math.log10(count + 1) * 0.18),
        ease: "power2.out",
        onUpdate: () => {
          if (num.current) num.current.textContent = fmt.format(Math.round(o.v));
        },
      });
    },
    { scope: root, dependencies: [count, prefix] },
  );

  return (
    <div ref={root} className={`result ${leaked ? "is-leaked" : "is-safe"}`}>
      <div className="result-icon" aria-hidden>
        {leaked ? "!" : "✓"}
      </div>
      <span ref={num} className="result-count">
        0
      </span>
      <span className="result-unit">{leaked ? (count === 1 ? "breach appearance" : "breach appearances") : "breach appearances"}</span>
      <h2>{leaked ? "This password has been leaked" : "Not found in any known breach"}</h2>
      <p>
        {leaked
          ? "Attackers try leaked passwords first. If you use this one anywhere, change it there now and don’t reuse it."
          : "Good news. That doesn’t guarantee it’s strong, so keep it long and unique to one account."}
      </p>
      <p className="sent">
        Sent over the network: <code>{prefix}</code> <span>· 5 of 40 hash characters</span>
      </p>
    </div>
  );
}

export default function App() {
  useLenis();
  const [password, setPassword] = useState("");
  const [reveal, setReveal] = useState(false);
  const [state, setState] = useState<State>({ kind: "idle" });
  const [total, setTotal] = useState<number | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const heroRef = useRef<HTMLElement>(null);
  const reduced = prefersReducedMotion();

  const loadStats = () =>
    fetch("/api/stats", { cache: "no-store" })
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => j && setTotal(j.totalChecks))
      .catch(() => {});

  useEffect(() => {
    loadStats();
  }, []);

  // Hero entrance (headline is animated by SplitText).
  useGSAP(
    () => {
      if (reduced) return;
      gsap
        .timeline({ defaults: { ease: "power3.out" } })
        .from(".nav", { y: -16, opacity: 0, duration: 0.6 })
        .from(".badge", { y: 10, opacity: 0, duration: 0.5 }, "<0.1")
        .from(".lede", { y: 14, opacity: 0, duration: 0.7 }, "+=0.55")
        .from(".checker", { y: 20, opacity: 0, duration: 0.7 }, "<0.12")
        .from(".trust li", { y: 8, opacity: 0, stagger: 0.07, duration: 0.45 }, "<0.25");
    },
    { scope: heroRef },
  );

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
    <>
      <div className="aurora" aria-hidden>
        {!reduced && (
          <SoftAurora
            color1="#9fe3d4"
            color2="#b8c2fb"
            brightness={0.9}
            opacity={0.55}
            lightMode
            speed={0.35}
            bandHeight={0.42}
            enableMouseInteraction={false}
            clickPulse={false}
          />
        )}
      </div>

      <div className="shell">
        <section className="hero" ref={heroRef}>
          <nav className="nav" aria-label="Primary">
            <a href="#top" className="brand" id="top">
              <span className="brand-mark" aria-hidden />
              LeakCheck
            </a>
            <div className="nav-links">
              <a href="#how">How it works</a>
            </div>
          </nav>

          <div className="badge">
            <span className="dot" aria-hidden />
            <ShinyText text="Free · No accounts · Private by design" color="#5d6884" shineColor="#1d6f63" speed={3} />
          </div>

          <SplitText
            text="Has your password been leaked?"
            tag="h1"
            className="headline"
            splitType="words"
            effect="rise"
            delay={70}
            duration={1}
            trigger="load"
            textAlign="center"
          />

          <p className="lede">
            Check any password against more than a billion exposed in real data breaches. It’s hashed on your device
            and never leaves your browser.
          </p>

          <form className="checker" onSubmit={onSubmit} autoComplete="off">
            <label htmlFor="pw" className="sr-only">
              Password to check
            </label>
            <div className="field">
              <svg className="lock" viewBox="0 0 24 24" aria-hidden>
                <rect x="4" y="10.5" width="16" height="10" rx="2.5" />
                <path d="M8 10.5V7.5a4 4 0 0 1 8 0v3" />
              </svg>
              <input
                id="pw"
                type={reveal ? "text" : "password"}
                value={password}
                onChange={(e) => onChange(e.target.value)}
                placeholder="Enter a password to check"
                autoComplete="off"
                autoCorrect="off"
                autoCapitalize="off"
                spellCheck={false}
                data-lpignore="true"
                data-1p-ignore
                autoFocus
              />
              <button type="button" className="toggle" onClick={() => setReveal((r) => !r)} aria-pressed={reveal}>
                {reveal ? "Hide" : "Show"}
              </button>
            </div>
            <button type="submit" className="primary" disabled={!password || state.kind === "loading"}>
              {state.kind === "loading" ? <span className="spinner" aria-label="Checking" /> : "Check password"}
            </button>
          </form>

          <div className="result-slot" aria-live="polite">
            {state.kind === "result" && <ResultCard key={`${state.prefix}-${state.count}`} {...state} />}
            {state.kind === "error" && (
              <div className="result is-error">
                <h2>Couldn’t check right now</h2>
                <p>{state.message}</p>
              </div>
            )}
          </div>

          <ul className="trust">
            <li>Hashed locally with Web Crypto</li>
            <li>Nothing stored or logged</li>
            <li>Powered by Have I Been Pwned</li>
          </ul>
        </section>

        <section id="how" className="section">
          <AnimatedContent distance={40} duration={0.9} threshold={0.15}>
            <p className="eyebrow">How it works</p>
            <h2 className="section-title">Your password stays yours</h2>
            <p className="section-lede">
              We use the k-anonymity model, so neither we nor Have I Been Pwned ever see what you typed.
            </p>
          </AnimatedContent>
          <div className="grid">
            {STEPS.map((s, i) => (
              <AnimatedContent key={s.n} distance={50} duration={0.8} delay={i * 0.12} threshold={0.1}>
                <SpotlightCard className="card" spotlightColor="rgba(45, 166, 145, 0.14)" theme="light">
                  <span className="card-num">{s.n}</span>
                  <h3>{s.title}</h3>
                  <p>{s.body}</p>
                </SpotlightCard>
              </AnimatedContent>
            ))}
          </div>
        </section>

        <footer className="footer">
          {total !== null && (
            <p className="stat">
              <strong>{fmt.format(total)}</strong> passwords checked privately
            </p>
          )}
          <p>
            Breach data from{" "}
            <a href="https://haveibeenpwned.com/Passwords" target="_blank" rel="noreferrer">
              Have I Been Pwned
            </a>{" "}
            · Free forever
          </p>
        </footer>
      </div>
    </>
  );
}
