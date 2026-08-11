#!/usr/bin/env bun

function estimateTokens(data: unknown): number {
  return Math.ceil(JSON.stringify(data).length / 4)
}

const fullDataset = Array.from({ length: 100 }, (_, index) => ({
  id: `post_${index}`,
  likesCount: (index * 137) % 5000,
  timestamp: index,
}))

const filtered = fullDataset
  .filter((post) => post.likesCount > 1000)
  .sort((a, b) => b.likesCount - a.likesCount)
  .slice(0, 10)

console.log(JSON.stringify({
  fullItems: fullDataset.length,
  filteredItems: filtered.length,
  estimatedFullTokens: estimateTokens(fullDataset),
  estimatedFilteredTokens: estimateTokens(filtered),
}, null, 2))
