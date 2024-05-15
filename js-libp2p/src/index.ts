import { enable } from '@libp2p/logger'
import delay from 'delay'
import { Heartbeat } from './heartbeat.js'
import { createLibp2p } from './libp2p.js'
import { createServer } from './server.js'

if (process.env.PARSEC_DEBUG != null) {
  enable('parsec*')
}

const libp2p = await createLibp2p()
const log = libp2p.logger.forComponent('parsec:server')
const server = await createServer(libp2p)

const address = await server.listen({
  host: '0.0.0.0',
  port: parseInt(process.env.PARSEC_SERVER_SERVER_PORT ?? '5001')
})

log('server listening on %s', address)
log('peer id is %p', libp2p.peerId)

if (process.env.PARSEC_SERVER_STARTUP_DELAY != null) {
  const startUpDelay = parseInt(process.env.PARSEC_SERVER_STARTUP_DELAY) * 1000
  await delay(startUpDelay)
}

// @ts-expect-error routingTable.size is not part of the interface
log('server ready, routing table size %d', libp2p.services.dht.routingTable.size)

const heartbeat = new Heartbeat(server.pg, libp2p)
await heartbeat.online()

// update offline time on graceful shutdown
process.on('SIGTERM', () => {
  log('server stopping')
  server.close()
    .then(() => {
      log('server stopped')
    })
    .catch(err => {
      log.error('error stopping server', err)
    })

  // may complete before server stops 🤷
  heartbeat.offline()
    .then(() => {
      process.exit(0)
    })
    .catch(() => {
      process.exit(1)
    })
})
