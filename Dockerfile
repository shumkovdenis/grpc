# Start by building the application.
FROM golang:1.20-bullseye AS build

WORKDIR /go/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /go/bin/app ./

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian11:nonroot

COPY --from=build /go/bin/app /

ENTRYPOINT ["/app"]
