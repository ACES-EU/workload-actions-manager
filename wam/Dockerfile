# syntax=docker/dockerfile:1

############################
# BUILD
############################
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY db db
COPY wam wam

RUN cd wam && CGO_ENABLED=0 go build \
  -mod=vendor \
  -a -o build/wam \
  ./cmd/wam

############################
# FINAL
############################
FROM alpine

COPY --from=builder /app/wam/build/wam /bin/wam

CMD [ "/bin/wam" ]
