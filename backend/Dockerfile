FROM golang:1.22-alpine AS backend-build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/cmd/ ./cmd/
RUN go build -o /out/agentbucket ./cmd/server/

FROM node:20-alpine AS frontend-build
WORKDIR /src
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
COPY index.html vite.config.js tailwind.config.js postcss.config.js ./
COPY src/ ./src/
RUN pnpm build

FROM alpine:3.20
RUN apk add --no-cache ca-certificates docker-cli git
COPY --from=backend-build /out/agentbucket /usr/local/bin/agentbucket
COPY --from=backend-build /src/cmd/sidecar/main.go /app/cmd/sidecar/main.go
COPY --from=frontend-build /src/dist /app/dist
WORKDIR /app
ENV AGENTBUCKET_ADDR=0.0.0.0:8080
ENV AGENTBUCKET_DATA_DIR=/data
ENV AGENTBUCKET_SIDECAR_HOST=host.docker.internal
ENV AGENTBUCKET_BUILD_TIMEOUT=300s
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/agentbucket"]
