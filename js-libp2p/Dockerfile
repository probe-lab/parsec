FROM --platform=$BUILDPLATFORM node:20-slim as builder

# Install dependencies required for building the app
RUN apt-get update && \
    apt-get install -y build-essential wget tini && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package*.json ./
RUN npm ci --quiet

COPY . .
RUN npm run build
RUN npm prune --omit=dev

# Create lightweight container
FROM --platform=$BUILDPLATFORM node:20-slim as app

ENV NODE_ENV production
WORKDIR /app

# copy built application from the builder stage
COPY --from=builder /app ./

# copy tini from the builder stage
COPY --from=builder /usr/bin/tini /usr/bin/tini

# create the parsec binary
RUN npm link

# port for HTTP API
EXPOSE 7070

# set up permissions for non-privleged user
RUN chown -R node:node /app

# run using a non-privleged user
USER node

# Use tini to handle signals properly, see https://github.com/nodejs/docker-node/blob/main/docs/BestPractices.md#handling-kernel-signals
ENTRYPOINT ["/usr/bin/tini", "-p", "SIGKILL", "--"]

CMD [ "parsec", "server" ]

# for best practices, see:
# * https://snyk.io/blog/10-best-practices-to-containerize-nodejs-web-applications-with-docker/
# * https://github.com/nodejs/docker-node/blob/main/docs/BestPractices.md
