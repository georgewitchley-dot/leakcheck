import { useEffect } from "react";

const isField = (t: EventTarget | null) =>
  t instanceof HTMLInputElement || t instanceof HTMLTextAreaElement;

/**
 * Locks the page against copying, cutting, selecting, dragging and the context menu.
 * Pasting into the password field is still allowed so password managers keep working,
 * but text can never be copied or cut *out* of it.
 */
export function usePageLock() {
  useEffect(() => {
    const block = (e: Event) => {
      e.preventDefault();
      e.stopPropagation();
    };
    const onSelectStart = (e: Event) => {
      if (!isField(e.target)) block(e);
    };
    const onPaste = (e: Event) => {
      if (!isField(e.target)) block(e);
    };
    const onKey = (e: KeyboardEvent) => {
      const mod = e.ctrlKey || e.metaKey;
      if (!mod) return;
      const k = e.key.toLowerCase();
      // Copy / cut / select-all outside the field, save page, print, view source.
      if (k === "c" || k === "x" || k === "s" || k === "p" || k === "u") block(e);
      if (k === "a" && !isField(e.target)) block(e);
    };

    const opts = { capture: true } as const;
    document.addEventListener("copy", block, opts);
    document.addEventListener("cut", block, opts);
    document.addEventListener("contextmenu", block, opts);
    document.addEventListener("dragstart", block, opts);
    document.addEventListener("drop", block, opts);
    document.addEventListener("selectstart", onSelectStart, opts);
    document.addEventListener("paste", onPaste, opts);
    document.addEventListener("keydown", onKey, opts);
    return () => {
      document.removeEventListener("copy", block, opts);
      document.removeEventListener("cut", block, opts);
      document.removeEventListener("contextmenu", block, opts);
      document.removeEventListener("dragstart", block, opts);
      document.removeEventListener("drop", block, opts);
      document.removeEventListener("selectstart", onSelectStart, opts);
      document.removeEventListener("paste", onPaste, opts);
      document.removeEventListener("keydown", onKey, opts);
    };
  }, []);
}
