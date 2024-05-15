import { getContext } from './context.js'
import type { PostgresDb } from '@fastify/postgres'
import type { Libp2p, Logger, PeerId } from '@libp2p/interface'

export class Heartbeat {
  private readonly db: PostgresDb
  private readonly peerId: PeerId
  private readonly log: Logger
  private row?: number
  private interval?: ReturnType<typeof setInterval>

  constructor (db: PostgresDb, libp2p: Libp2p) {
    this.db = db
    this.peerId = libp2p.peerId
    this.log = libp2p.logger.forComponent('parsec:heartbeat')
  }

  async online (): Promise<void> {
    const context = getContext()
    const queryArgs = {
      cpu: context.cpus,
      memory: context.memory,
      peer_id: this.peerId.toString(),
      region: process.env.AWS_REGION ?? 'unknown',
      cmd: process.argv.join(' '),
      fleet: process.env.PARSEC_SERVER_FLEET ?? 'unknown',
      dependencies: '{}',
      ip_address: context.ipAddress,
      server_port: process.env.PARSEC_SERVER_SERVER_PORT,
      peer_port: 0,
      last_heartbeat: new Date().toISOString(),
      created_at: new Date().toISOString()
    }
    const queryKeys = Object.keys(queryArgs)
    const queryValues = Object.values(queryArgs)
    const query = `INSERT INTO nodes_ecs(${queryKeys.join(', ')}) VALUES(${queryKeys.map((_, index) => `$${index + 1}`).join(', ')}) RETURNING "id"`
    const result = await this.db.query(query, queryValues)

    this.row = result.rows[0].id

    this.interval = setInterval(() => {
      this.update()
        .catch(err => {
          this.log.error('could not update heartbeat', err)
        })
    }, 60000)
  }

  async offline (): Promise<void> {
    clearInterval(this.interval)

    const queryArgs = {
      offline_since: new Date().toISOString()
    }

    const queryValues = Object.values(queryArgs)
    const query = `UPDATE nodes_ecs SET ${Object.entries(queryArgs).map(([key], index) => `${key} = $${index + 1}`).join(', ')} WHERE id = $${queryValues.length + 1}`

    this.log('set offline', this.row)

    await this.db.query(query, [
      ...queryValues,
      this.row
    ])
  }

  private async update (): Promise<void> {
    const queryArgs = {
      last_heartbeat: new Date().toISOString()
    }

    const queryValues = Object.values(queryArgs)
    const query = `UPDATE nodes_ecs SET ${Object.entries(queryArgs).map(([key], index) => `${key} = $${index + 1}`).join(', ')} WHERE id = $${queryValues.length + 1}`

    this.log('update heartbeat', this.row)

    await this.db.query(query, [
      ...queryValues,
      this.row
    ])
  }
}
