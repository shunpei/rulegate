import type { AskRequest, AskResponse, ErrorResponse } from "./types";

const API_BASE = "/api";

export class ApiError extends Error {
  constructor(
    public status: number,
    public body: ErrorResponse,
  ) {
    super(body.error);
  }
}

export async function askQuestion(req: AskRequest): Promise<AskResponse> {
  const res = await fetch(`${API_BASE}/ask`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });

  if (!res.ok) {
    const body: ErrorResponse = await res.json();
    throw new ApiError(res.status, body);
  }

  return res.json();
}
