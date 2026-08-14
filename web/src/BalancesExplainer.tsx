import { useEffect, useState } from "react";
import { api } from "./api/client";
import { Glossary } from "./Glossary";

type Balances = { ledger: string; holds: string; available: string };

export function BalancesExplainer({ accountId, token }: { accountId: string; token: string }) {
  const [balances, setBalances] = useState<Balances | null>(null);

  useEffect(() => {
    api<Balances>(`/accounts/${accountId}/balances`, token).then(setBalances);
  }, [accountId, token]);

  if (!balances) return <p>Loading…</p>;

  const hasHold = balances.holds !== "0.00 IDR";

  return (
    <div>
      <h3>Balances</h3>
      <div style={{ display: "flex", gap: 24 }}>
        <div>
          <Glossary term="Ledger balance">
            <strong>Ledger</strong>
          </Glossary>
          <div>{balances.ledger}</div>
        </div>
        <div>
          <Glossary term="Available balance">
            <strong>Available</strong>
          </Glossary>
          <div>{balances.available}</div>
        </div>
      </div>
      {hasHold && (
        <p>
          <Glossary term="Hold">Available</Glossary> is {balances.holds} lower than Ledger because
          of an active hold reserving funds that have not posted yet.
        </p>
      )}
    </div>
  );
}
