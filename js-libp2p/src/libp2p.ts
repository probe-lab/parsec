import { noise } from '@chainsafe/libp2p-noise'
import { yamux } from '@chainsafe/libp2p-yamux'
import { bootstrap } from '@libp2p/bootstrap'
import { identify } from '@libp2p/identify'
import { kadDHT, removePrivateAddressesMapper } from '@libp2p/kad-dht'
import { tcp } from '@libp2p/tcp'
import { tls } from '@libp2p/tls'
import { webSockets } from '@libp2p/websockets'
import { LevelDatastore } from 'datastore-level'
import { createLibp2p as createNode } from 'libp2p'
import type { Libp2p } from '@libp2p/interface'
import type { KadDHT } from '@libp2p/kad-dht'

export async function createLibp2p (): Promise<Libp2p<{ dht: KadDHT }>> {
  const path = process.env.PARSEC_SERVER_LEVELDB ?? 'datastore.db'
  const datastore = new LevelDatastore(path)
  await datastore.open()

  return createNode({
    datastore,
    addresses: {
      announce: [
        '/dns4/example.com/tcp/1234'
      ]
    },
    transports: [
      tcp(),
      webSockets()
    ],
    connectionEncryption: [
      noise(),
      tls()
    ],
    streamMuxers: [
      yamux()
    ],
    peerDiscovery: [
      bootstrap({
        list: [
          '/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN',
          '/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa',
          '/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb',
          '/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt',
          '/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ'
        ]
      })
    ],
    services: {
      identify: identify(),
      dht: kadDHT({
        protocol: '/ipfs/kad/1.0.0',
        peerInfoMapper: removePrivateAddressesMapper
      })
    }
  })
}
