import { CID } from 'multiformats/cid'
import * as raw from 'multiformats/codecs/raw'
import { sha256 } from 'multiformats/hashes/sha2'
import { fromString as uint8ArrayFromString } from 'uint8arrays/from-string'
import { toNanoSeconds, toSeconds } from './utils.js'
import type { Libp2p, Logger } from '@libp2p/interface'
import type { KadDHT } from '@libp2p/kad-dht'
import type { FastifyRequest, FastifyReply } from 'fastify'

interface PostProvideRequestBody {
  Content: string
  Target?: 'DHT' | 'IPNI'
}

interface PostRetrieveByCidRequestBody {
  cid: string
}

export class Service {
  private readonly libp2p: Libp2p<{ dht: KadDHT }>
  private readonly log: Logger

  constructor (libp2p: Libp2p<{ dht: KadDHT }>) {
    this.libp2p = libp2p
    this.log = libp2p.logger.forComponent('parsec:service')
  }

  // Operation: postProvide
  // URL: /provide
  // summary:  Publishes a provider record for the given content.
  // req.headers
  //   type: object
  //   properties:
  //     x-scheduler-id:
  //       type: string
  //       description: >-
  //         An identifier of the scheduler that's doing the request. This value is
  //         used for prometheus metrics.
  //
  // req.body
  //   required:
  //     - Content
  //   properties:
  //     Content:
  //       type: string
  //       format: base64
  //       description: >
  //         Binary data the server should use to generate a CID and publish a provider
  //         record for.
  //
  //         The Go implementation generates 1024 bytes of random data and puts into
  //         this JSON field.
  //     Target:
  //       type: string
  //       enum:
  //         - DHT
  //         - IPNI
  //       default: DHT
  //       description: >
  //         Specifies the provide target type. If set to DHT (default when property !=
  //         IPNI) the server
  //
  //         will write provider records to the DHT. If set to IPNI, the server
  //         announces an advertisement
  //
  //         to an InterPlanetary Network Indexer. To which specifically is part of the
  //         servers configuration
  //
  //         and the client must know how the server is configured to know the specific
  //         IPNI (e.g, whether
  //
  //         it's cid.contact or another one)
  //
  // valid responses
  //   '200':
  //     description: >
  //       The result of the provider record publication. Any error that might have
  //       happened during that
  //
  //       process should be passed to the `Error` field. For the sake of the
  //       measurement we still consider
  //
  //       an erroneous publication a valid data point. Hence, a `200` status code.
  //     content:
  //       application/json:
  //         schema:
  //           required:
  //             - CID
  //             - Duration
  //             - Error
  //             - RoutingTableSize
  //           properties:
  //             CID:
  //               type: string
  //               format: CID
  //               example: bafybeihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku
  //             Duration:
  //               type: integer
  //               description: >-
  //                 The time it took to write all provider records in nanoseconds
  //                 (default Go formatting of `time.Duration`). E.g., `1000000000` for
  //                 1s.
  //               example: 1000000000
  //             Error:
  //               type: string
  //               description: >-
  //                 Just any text that indicates the error reason. If no error
  //                 happened, pass an empty string.
  //             RoutingTableSize:
  //               type: integer
  //               description: >-
  //                 The number of peers in the routing table. Either right before or
  //                 right after the publication. Doesn't really matter.
  //               example: 202
  //   '400':
  //     description: E.g., the given JSON was malformed.
  //

  async postProvide (req: FastifyRequest<{ Body: PostProvideRequestBody }>, reply: FastifyReply): Promise<void> {
    this.log('POST /provide')

    try {
      const { Content, Target } = req.body

      // no IPNI support yet
      if (Target != null && Target !== 'DHT') {
        return await reply.code(400).send()
      }

      const buf = uint8ArrayFromString(Content, 'base64')
      const digest = await sha256.digest(buf)
      const cid = CID.createV1(raw.code, digest)

      const start = Date.now()
      let duration = 0
      let error: string | undefined
      let providers = 0

      try {
        // @ts-expect-error routingTable.size is not part of the interface
        this.log('start publishing provider records cid=%c rtsize=%d', cid, this.libp2p.services.dht.routingTable.size)

        for await (const event of this.libp2p.services.dht.provide(cid)) {
          if (event.name === 'PEER_RESPONSE' && event.messageName === 'ADD_PROVIDER') {
            providers++
          }
        }

        duration = Date.now() - start

        // @ts-expect-error routingTable.size is not part of the interface
        this.log('finished publishing provider records cid=%c dur=%d rtsize=%d', cid, toSeconds(duration), this.libp2p.services.dht.routingTable.size)
      } catch (err: any) {
        error = err.toString()

        // @ts-expect-error routingTable.size is not part of the interface
        this.log.error('error publishing provider records cid=%c rtsize=%d', cid, this.libp2p.services.dht.routingTable.size, err)
      }

      if (providers === 0) {
        error = 'Did not publish to any providers'
      }

      return await reply.code(200)
        .header('content-type', 'application/json')
        .send(
          JSON.stringify({
            CID: cid.toString(),
            Duration: toNanoSeconds(duration),

            // @ts-expect-error routingTable.size is not part of the interface
            RoutingTableSize: this.libp2p.services.dht.routingTable.size,
            Error: error
          })
        )
    } catch (err) {
      this.log.error(err)
      return reply.code(500).send()
    }
  }

  // Operation: postRetrieveByCid
  // URL: /retrieve/:cid
  // summary:  Looks up provider records for the given CID.
  // req.headers
  //   type: object
  //   properties:
  //     x-scheduler-id:
  //       type: string
  //       description: >-
  //         An identifier of the scheduler that's doing the request. This value is
  //         used for prometheus metrics.
  //
  // req.params
  //   type: object
  //   properties:
  //     cid:
  //       type: string
  //       description: CID to look up
  //   required:
  //     - cid
  //
  // req.body
  //   type: object
  //
  // valid responses
  //   '200':
  //     description: >
  //       The result of the provider record look up. Any error that might have
  //       happened during that
  //
  //       process should be passed to the `Error` field. For the sake of the
  //       measurement we still consider
  //
  //       an erroneous retrieval a valid data point. Hence, a `200` status code.
  //     content:
  //       application/json:
  //         schema:
  //           required:
  //             - CID
  //             - Duration
  //             - Error
  //             - RoutingTableSize
  //           properties:
  //             CID:
  //               type: string
  //               format: CID
  //               example: bafybeihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku
  //             Duration:
  //               type: integer
  //               description: >-
  //                 The time it took to **find the first** provider records in
  //                 nanoseconds (default Go formatting of `time.Duration`). E.g.,
  //                 `1000000000` for 1s.
  //               example: 1000000000
  //             Error:
  //               type: string
  //               description: >
  //                 Just any text that indicates the error reason. If no error
  //                 happened, pass an empty string.
  //
  //                 If the lookup algorithm couldn't find a provider record but didn't
  //                 really encounter
  //
  //                 an error, this field should be mapped to the value `not found`.
  //             RoutingTableSize:
  //               type: integer
  //               description: >-
  //                 The number of peers in the routing table. Either right before or
  //                 right after the publication. Doesn't really matter.
  //               example: 202
  //   '400':
  //     description: E.g., the JSON is malformed or we couldn't parse the given CID.
  //

  async postRetrieveByCid (req: FastifyRequest<{ Params: PostRetrieveByCidRequestBody } >, reply: FastifyReply): Promise<void> {
    this.log('POST /retrieve/:cid')

    const cid = CID.parse(req.params.cid)
    const start = Date.now()
    let duration = 0
    let error: string | undefined

    try {
      // @ts-expect-error routingTable.size is not part of the interface
      this.log('start finding providers cid=%c rtsize=%d', cid, this.libp2p.services.dht.routingTable.size)
      let foundProvider = false

      for await (const event of this.libp2p.services.dht.findProviders(cid)) {
        if (event.name === 'PEER_RESPONSE' && event.providers.length > 0) {
          foundProvider = true
          duration = Date.now() - start

          // @ts-expect-error routingTable.size is not part of the interface
          this.log('found provider cid=%c dur=%d provider=%p rtsize=%d', cid, toSeconds(duration), event.providers[0]?.id, this.libp2p.services.dht.routingTable.size)
          break
        }
      }

      if (!foundProvider) {
        error = 'Did not find any providers'
        // @ts-expect-error routingTable.size is not part of the interface
        this.log('did not find any providers cid=%c dur=%d provider=%p rtsize=%d', cid, toSeconds(duration), this.libp2p.services.dht.routingTable.size)
      }
    } catch (err: any) {
      this.log.error(err)
      error = err.toString()

      // @ts-expect-error routingTable.size is not part of the interface
      this.log.error('error finding providers cid=%c rtsize=%d', cid, this.libp2p.services.dht.routingTable.size, err)
    }

    return reply.code(200).send(
      JSON.stringify({
        CID: cid.toString(),
        Duration: toNanoSeconds(duration),

        // @ts-expect-error routingTable.size is not part of the interface
        RoutingTableSize: this.libp2p.services.dht.routingTable.size,
        Error: error
      })
    )
  }

  // Operation: getReadiness
  // URL: /readiness
  // summary:  Indicates readiness for accepting publication or retrieval requests.
  // valid responses
  //   '200':
  //     description: The server is ready to accept publication or retrieval requests.
  //

  async getReadiness (req: FastifyRequest, reply: FastifyReply): Promise<void> {
    this.log('GET /readiness')
    return reply.code(200).send(' ') // content-free 200 is 201
  }
}
