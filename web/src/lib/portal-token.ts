export interface PortalSession {
  token: string;
  consumer: string;
  expiresAt: Date | null;
}

function decodeBase64URL(input: string): string {
  const normalized = input.replace(/-/g, "+").replace(/_/g, "/");
  const padded = normalized + "=".repeat((4 - (normalized.length % 4)) % 4);
  return atob(padded);
}

export function parsePortalToken(token: string): PortalSession | null {
  if (!token) return null;

  const parts = token.split(".");
  let consumerPart = "";
  let expPart = "";

  if (parts.length === 5 && parts[0] === "spt_v2") {
    consumerPart = parts[2];
    expPart = parts[3];
  } else {
    return null;
  }

  let consumer = "";
  try {
    consumer = decodeBase64URL(consumerPart);
  } catch {
    return null;
  }

  const expUnix = Number(expPart);
  return {
    token,
    consumer,
    expiresAt: Number.isFinite(expUnix) ? new Date(expUnix * 1000) : null,
  };
}
