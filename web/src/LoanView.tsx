import { useState } from "react";
import { api } from "./api/client";
import { Glossary } from "./Glossary";

type Loan = {
  id: string;
  deposit_account_id: string;
  principal: string;
  term_months: number;
  interest_rate_bps: number;
  status: string;
  disbursed_at: string;
};

type Installment = {
  number: number;
  due_date: string;
  principal: string;
  interest: string;
  total: string;
  balance: string;
  status: string;
};

export function LoanView({ token }: { token: string }) {
  const [depositAccountId, setDepositAccountId] = useState("");
  const [principal, setPrincipal] = useState("12000000.00");
  const [termMonths, setTermMonths] = useState(12);
  const [loan, setLoan] = useState<Loan | null>(null);
  const [schedule, setSchedule] = useState<Installment[]>([]);
  const [error, setError] = useState<string | null>(null);

  async function loadSchedule(loanId: string) {
    setSchedule(await api<Installment[]>(`/loans/${loanId}/schedule`, token));
  }

  async function originate(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      const l = await api<Loan>("/loans", token, {
        method: "POST",
        body: JSON.stringify({
          product_code: "PL",
          deposit_account_id: depositAccountId,
          principal,
          term_months: termMonths,
        }),
      });
      setLoan(l);
      await loadSchedule(l.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "could not originate loan");
    }
  }

  async function repay() {
    if (!loan) return;
    setError(null);
    try {
      await api(`/loans/${loan.id}/repay`, token, { method: "POST" });
      await loadSchedule(loan.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "repayment failed");
    }
  }

  return (
    <div>
      <h3>Loans</h3>
      <p>
        <Glossary term="Amortization">A loan</Glossary> pays the same total installment every month.
        Early installments are mostly interest on the large remaining balance; later ones are mostly
        principal, until the last installment pays the balance to exactly zero.
      </p>
      {!loan && (
        <form onSubmit={originate}>
          <div>
            <label>
              Deposit account ID (disburses into it, repays out of it)
              <input value={depositAccountId} onChange={(e) => setDepositAccountId(e.target.value)} />
            </label>
          </div>
          <div>
            <label>
              Principal
              <input value={principal} onChange={(e) => setPrincipal(e.target.value)} />
            </label>
          </div>
          <div>
            <label>
              Term (months)
              <input
                type="number"
                min={1}
                value={termMonths}
                onChange={(e) => setTermMonths(Number(e.target.value))}
              />
            </label>
          </div>
          <button type="submit">Originate loan (product PL)</button>
        </form>
      )}
      {error && <p style={{ color: "crimson" }}>{error}</p>}
      {loan && (
        <div>
          <p>
            Loan {loan.id.slice(0, 8)}: {loan.principal} over {loan.term_months} months at{" "}
            {loan.interest_rate_bps / 100}% p.a., status {loan.status}.
          </p>
          <button onClick={repay} disabled={loan.status === "closed"}>
            Pay next installment
          </button>
          <table>
            <thead>
              <tr>
                <th>#</th>
                <th>Due</th>
                <th>Principal</th>
                <th>Interest</th>
                <th>Total</th>
                <th>Balance</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {schedule.map((inst) => (
                <tr key={inst.number}>
                  <td>{inst.number}</td>
                  <td>{inst.due_date}</td>
                  <td>{inst.principal}</td>
                  <td>{inst.interest}</td>
                  <td>{inst.total}</td>
                  <td>{inst.balance}</td>
                  <td>{inst.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
