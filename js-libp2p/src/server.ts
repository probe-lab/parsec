// Fastify plugin autogenerated by fastify-openapi-glue
import fastifyPostgres from '@fastify/postgres'
import fastify, { type FastifyInstance } from 'fastify'
import openapiGlue from 'fastify-openapi-glue'
import { Security } from './security.js'
import { Service } from './service.js'
import type { Libp2p } from '@libp2p/interface'
import type { KadDHT } from '@libp2p/kad-dht'

const localFile = (fileName: string): string => new URL(fileName, import.meta.url).pathname

export async function createServer (libp2p: Libp2p<{ dht: KadDHT }>): Promise<FastifyInstance> {
  const pluginOptions = {
    specification: localFile('./openApi.json'),
    serviceHandlers: new Service(libp2p),
    securityHandlers: new Security()
  }

  const server = fastify({
    ajv: {
      customOptions: {
        strict: false
      }
    }
  })

  // configure pg plugin if ENV vars are present
  if (process.env.PARSEC_DATABASE_USER != null && process.env.PARSEC_DATABASE_PASSWORD != null && process.env.PARSEC_DATABASE_HOST != null && process.env.PARSEC_DATABASE_NAME != null) {
    let connectionString = `postgres://${process.env.PARSEC_DATABASE_USER}:${process.env.PARSEC_DATABASE_PASSWORD}@${process.env.PARSEC_DATABASE_HOST}`

    if (process.env.PARSEC_DATABASE_PORT != null) {
      connectionString += `:${process.env.PARSEC_DATABASE_PORT}`
    }

    connectionString += `/${process.env.PARSEC_DATABASE_NAME}`

    if (process.env.PARSEC_DATABASE_SSL_MODE != null) {
      connectionString += `?sslmode=${process.env.PARSEC_DATABASE_SSL_MODE}`
    }

    await server.register(fastifyPostgres, {
      connectionString
    })
  }

  await server.register(openapiGlue, pluginOptions)

  return server
}
