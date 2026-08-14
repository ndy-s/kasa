import { useEffect, useState } from "react";
import { api } from "./api/client";

type Line = { account_id: string; direction: string; amount: string };
type Journal = { entry_id: string; type: string; lines: Line[]; balanced: boolean };

export function JournalInspector({ entryId, token }: { entryId: string; token: string }) {
  const [journal, setJournal] = useState<Journal | null>(null);

  useEffect(() => {
    api<Journal>(`/transactions/${entryId}/journal`, token).then(setJournal);
  }, [entryId, token]);

  if (!journal) return <p>Loading…</p>;

  const debits = journal.lines.filter((l) => l.direction === "debit");
  const credits = journal.lines.filter((l) => l.direction === "credit");

  return (
    <div>
      <h3>{journal.type}</h3>
      <div style={{ display: "flex", gap: 24 }}>
        <div>
          <strong>Debits</strong>
          {debits.map((l, i) => <div key={i}>{l.amount}</div>)}
        </div>
        <div>
          <strong>Credits</strong>
          {credits.map((l, i) => <div key={i}>{l.amount}</div>)}
        </div>
      </div>
      <p>{journal.balanced ? "debits = credits ✓" : "unbalanced ✗"}</p>
    </div>
  );
}
