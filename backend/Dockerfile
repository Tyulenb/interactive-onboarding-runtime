FROM golang:1.26.4 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY migrations/ ./migrations/
COPY fonts/ ./fonts/

RUN CGO_ENABLED=0 GOOS=linux go build -o onboarding-backend cmd/app/main.go

FROM alpine:3.23 AS prod

WORKDIR /app

RUN addgroup -S app && adduser -S -G app app

COPY --from=builder /app/onboarding-backend ./onboarding-backend
COPY --from=builder /app/fonts ./fonts/

USER app

EXPOSE 8080

CMD ["./onboarding-backend"]
