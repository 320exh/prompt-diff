async function request(path, opts) {
  const res = await fetch(path, opts)
  let body = null
  try {
    body = await res.json()
  } catch {
    // non-JSON response (e.g. exports); caller handles res directly
  }
  if (!res.ok) {
    const message = (body && body.error) || res.statusText || 'request failed'
    throw new Error(message)
  }
  return body
}

export function postJSON(path, payload) {
  return request(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

export function getJSON(path) {
  return request(path)
}

export const DEFAULT_PROMPT = `---
name: Customer Support Classifier
version: 2.0.0
models:
  - gpt-4o
  - claude-3-5-sonnet
variables:
  - user_context
---
You are an expert customer support agent.
Analyze the user query: {{ user_context }}

Return JSON with classification and confidence score.`
