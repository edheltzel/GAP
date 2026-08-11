import { NextResponse } from "next/server"

export async function POST(request: Request) {
  let message: unknown
  try {
    const body = await request.json() as { message?: unknown }
    message = body.message
  } catch {
    return NextResponse.json({ error: "Request body must be valid JSON" }, { status: 400 })
  }

  if (typeof message !== "string" || !message.trim()) {
    return NextResponse.json({ error: "Message is required" }, { status: 400 })
  }

  return NextResponse.json(
    {
      error: "Dashboard chat is disabled until the host application supplies a reviewed current model adapter.",
      adapter_called: false,
    },
    { status: 501 },
  )
}
