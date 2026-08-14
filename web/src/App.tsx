import { useState } from "react";
import { api } from "./api/client";
import { JournalInspector } from "./JournalInspector";
import { BalancesExplainer } from "./BalancesExplainer";
import "./App.css";

function LoginForm({ onLogin }: { onLogin: (token: string) => void }) {
  const [email, setEmail] = useState("demo@example.com");
  const [password, setPassword] = useState("demo-password");
  const [error, setError] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      const { token } = await api<{ token: string }>("/login", "", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      onLogin(token);
    } catch (err) {
      setError(err instanceof Error ? err.message : "login failed");
    }
  }

  return (
    <form onSubmit={submit}>
      <h1>Kasa</h1>
      <p>Sign in to inspect a transaction's journal and a balance's holds.</p>
      <div>
        <label>
          Email
          <input value={email} onChange={(e) => setEmail(e.target.value)} />
        </label>
      </div>
      <div>
        <label>
          Password
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </label>
      </div>
      {error && <p style={{ color: "crimson" }}>{error}</p>}
      <button type="submit">Log in</button>
    </form>
  );
}

function Explorer({ token }: { token: string }) {
  const [accountId, setAccountId] = useState("");
  const [entryId, setEntryId] = useState("");

  return (
    <div>
      <h1>Kasa explorer</h1>
      <div>
        <label>
          Account ID
          <input value={accountId} onChange={(e) => setAccountId(e.target.value)} />
        </label>
      </div>
      {accountId && <BalancesExplainer accountId={accountId} token={token} />}
      <div>
        <label>
          Transaction ID
          <input value={entryId} onChange={(e) => setEntryId(e.target.value)} />
        </label>
      </div>
      {entryId && <JournalInspector entryId={entryId} token={token} />}
    </div>
  );
}

function App() {
  const [token, setToken] = useState<string | null>(null);
  return token ? <Explorer token={token} /> : <LoginForm onLogin={setToken} />;
}

export default App;
