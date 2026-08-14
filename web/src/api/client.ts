const BASE = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export async function api<T>(path: string, token: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      ...(init?.headers ?? {}),
    },
  });
  if (!res.ok) {
    throw new Error(`${res.status} ${await res.text()}`);
  }
  return res.json() as Promise<T>;
}
