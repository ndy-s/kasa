import type { ReactNode } from "react";

const definitions: Record<string, string> = {
  "Ledger balance": "The sum of every posted journal line touching this account. What you truly have.",
  "Available balance": "Ledger balance minus active holds. What you can spend right now.",
  Hold: "A reserved amount, like a card authorization, that reduces available balance before it posts to the ledger.",
  "Journal entry": "A balanced set of debit and credit lines recorded together. Debits always equal credits.",
  Reversal: "A new entry with debits and credits swapped, compensating a prior entry without deleting it.",
  Idempotency: "A retried request with the same key returns the original result instead of repeating the action.",
  Amortization:
    "Splitting a loan into equal payments over time, each one part interest on the remaining balance and part principal.",
};

export function Glossary({ term, children }: { term: string; children: ReactNode }) {
  const definition = definitions[term];
  return (
    <span title={definition ?? term} style={{ borderBottom: "1px dotted", cursor: "help" }}>
      {children}
    </span>
  );
}
