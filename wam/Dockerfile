# syntax=docker/dockerfile:1

############################
# BUILD
############################
FROM golang:1.22.1-alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build \
  -mod=vendor \
  -a -o build/wam \
  ./cmd/wam

############################
# FINAL
############################
FROM alpine

COPY --from=builder /app/build/wam /bin/wam

CMD [ "/bin/wam" ]
