#!/usr/bin/env node

import { parseArgs } from 'node:util'

const args = parseArgs({
  options: {
    name: {
      type: 'string',
      short: 'n'
    },
    verbose: {
      type: 'boolean',
      short: 'v'
    }
  },
  allowPositionals: true
})

if (args.positionals[0] === 'server') {
  await import('./index.js')
} else {
  throw new Error('Only running `parsec server` is supported')
}
