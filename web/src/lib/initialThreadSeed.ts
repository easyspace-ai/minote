/** Dedupe auto-send of the first message when React Strict Mode double-mounts dev effects. */
const sent = new Set<string>();

function seedSignature(threadId: string, text: string, fileKey: string): string {
  return `${threadId}::${text}::${fileKey}`;
}

export function claimInitialThreadSeed(
  threadId: string,
  text: string,
  fileKey = "",
): boolean {
  const sig = seedSignature(threadId, text, fileKey);
  if (sent.has(sig)) return false;
  sent.add(sig);
  return true;
}

export function releaseInitialThreadSeed(
  threadId: string,
  text: string,
  fileKey = "",
) {
  sent.delete(seedSignature(threadId, text, fileKey));
}
