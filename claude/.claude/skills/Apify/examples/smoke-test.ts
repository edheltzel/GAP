#!/usr/bin/env bun

import { Apify, resolveApifyToken } from '../index.ts'

function withCredentialsCleared<T>(operation: () => T): T {
  const token = process.env.APIFY_TOKEN
  const apiKey = process.env.APIFY_API_KEY

  delete process.env.APIFY_TOKEN
  delete process.env.APIFY_API_KEY

  try {
    return operation()
  } finally {
    if (token === undefined) delete process.env.APIFY_TOKEN
    else process.env.APIFY_TOKEN = token

    if (apiKey === undefined) delete process.env.APIFY_API_KEY
    else process.env.APIFY_API_KEY = apiKey
  }
}

withCredentialsCleared(() => {
  for (const preflight of [() => resolveApifyToken(), () => new Apify()]) {
    try {
      preflight()
      throw new Error('missing credentials did not fail')
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      if (!message.includes('APIFY_TOKEN or APIFY_API_KEY')) throw error
    }
  }
})

const client = new Apify('offline-test-token')
if (!(client instanceof Apify)) throw new Error('Apify module did not construct')

console.log('PASS: Apify module loads and credential preflight stays offline')
