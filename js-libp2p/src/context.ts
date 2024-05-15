import os from 'node:os'

interface ECSContext {
  DockerId: string
  Name: string
  DockerName: string
  Image: string
  ImageID: string
  Labels: Record<string, string>
  DesiredStatus: string
  KnownStatus: string
  Limits: {
    CPU: number
    Memory: 128
  }
  CreatedAt: string
  StartedAt: string
  Type: string
  LogDriver: string
  LogOptions: Record<string, string>
  ContainerARN: string
  Networks: ESCNetwork[]
}

interface ESCNetwork {
  NetworkMode: string
  IPv4Addresses: string[]
  AttachmentIndex: number
  MACAddress: string
  IPv4SubnetCIDRBlock: string
  PrivateDNSName: string
  SubnetGatewayIpv4Address: string
}

export interface Context {
  cpus: number
  memory: number
  ipAddress: string
}

export function getContext (): Context {
  const awsEnvVar = process.env.ECS_CONTAINER_METADATA_URI_V4

  if (awsEnvVar != null) {
    const metadata: ECSContext = typeof awsEnvVar === 'string' ? JSON.parse(awsEnvVar) : awsEnvVar
    const network = metadata.Networks.find(net => net.NetworkMode === 'awsvpc')
    const ipAddress = network?.IPv4Addresses[0] ?? '0.0.0.0'

    return {
      cpus: metadata.Limits.CPU,
      memory: metadata.Limits.Memory,
      ipAddress
    }
  }

  const iface = os.networkInterfaces()

  return {
    cpus: os.cpus().length,
    memory: Math.round(os.totalmem() / 1024),
    ipAddress: iface.eth0?.[0].address ?? '127.0.0.1'
  }
}
